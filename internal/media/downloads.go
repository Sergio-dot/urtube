package media

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"github.com/google/uuid"
)

// subscriberBuffer is how many updates a slow subscriber may lag behind before
// progress updates are dropped for it.
const subscriberBuffer = 16

// DownloadService runs downloads in the background, at most maxConcurrent at a
// time in the order they were started, and publishes every state change to
// subscribers. All state is kept in memory: urtube runs as a single instance.
type DownloadService struct {
	fetcher       Fetcher
	destination   Destination
	maxConcurrent int

	mu          sync.Mutex
	jobs        map[string]*job // queued and running downloads
	pending     []*job          // queued downloads, oldest first
	running     int
	nextSeq     uint64
	subscribers map[chan Download]struct{}
	draining    bool
	closed      bool
	inFlight    sync.WaitGroup
}

type job struct {
	seq    uint64
	state  Download
	req    Request
	dir    string // where the finished files go, fixed when it was started
	ctx    context.Context
	cancel context.CancelFunc
}

// NewDownloadService creates a DownloadService that saves downloads where
// destination says and runs at most maxConcurrent at once; further downloads
// wait as StatusQueued.
func NewDownloadService(fetcher Fetcher, destination Destination, maxConcurrent int) *DownloadService {
	return &DownloadService{
		fetcher:       fetcher,
		destination:   destination,
		maxConcurrent: max(maxConcurrent, 1),
		jobs:          make(map[string]*job),
		subscribers:   make(map[chan Download]struct{}),
	}
}

// Start validates req and queues it for download, returning its initial state.
func (s *DownloadService) Start(req Request) (Download, error) {
	if err := req.Validate(); err != nil {
		return Download{}, err
	}
	// Checked up front so a misconfigured folder fails the request instead of
	// every queued download.
	dir, err := s.destination.DownloadDir()
	if err != nil {
		return Download{}, fmt.Errorf("%w: %w", ErrDownloadDirUnavailable, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.draining {
		return Download{}, ErrDraining
	}

	// The download outlives the request that started it.
	ctx, cancel := context.WithCancel(context.Background())
	j := &job{
		seq: s.nextSeq,
		state: Download{
			ID:      uuid.NewString(),
			VideoID: req.VideoID,
			Title:   req.Title,
			Status:  StatusQueued,
		},
		req:    req,
		dir:    dir,
		ctx:    ctx,
		cancel: cancel,
	}
	s.nextSeq++
	s.jobs[j.state.ID] = j
	s.pending = append(s.pending, j)
	// Add under the lock so it cannot race with Drain's Wait.
	s.inFlight.Add(1)
	s.publishLocked(j.state)
	initial := j.state

	s.dispatchLocked()
	return initial, nil
}

// dispatchLocked starts queued downloads while slots are free. s.mu must be
// held.
func (s *DownloadService) dispatchLocked() {
	for s.running < s.maxConcurrent && len(s.pending) > 0 {
		j := s.pending[0]
		s.pending = s.pending[1:]
		s.running++
		j.state.Status = StatusDownloading
		s.publishLocked(j.state)
		go s.run(j)
	}
}

func (s *DownloadService) run(j *job) {
	err := s.fetcher.Fetch(j.ctx, j.req, j.dir, func(p Progress) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if j.state.Status != StatusDownloading {
			return
		}
		j.state.Progress = p
		s.publishLocked(j.state)
	})

	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	// Check the context first: a cancelled fetch also returns an error.
	case j.ctx.Err() != nil:
		s.finishLocked(j, StatusCancelled, "")
	case err != nil:
		s.finishLocked(j, StatusError, err.Error())
	default:
		s.finishLocked(j, StatusFinished, "")
	}
}

// finishLocked moves j to a terminal status, frees its slot if it was running
// and starts the next queued download. A queued job must already be removed
// from s.pending. s.mu must be held.
func (s *DownloadService) finishLocked(j *job, status Status, errMsg string) {
	if j.state.Status == StatusDownloading {
		s.running--
	}
	j.state.Status = status
	j.state.Error = errMsg
	s.publishLocked(j.state)
	delete(s.jobs, j.state.ID)
	j.cancel()
	s.inFlight.Done()
	s.dispatchLocked()
}

// Cancel stops an active download. It returns ErrNotFound if id is unknown or
// the download has already ended.
func (s *DownloadService) Cancel(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	if !ok {
		return ErrNotFound
	}
	if j.state.Status == StatusQueued {
		s.pending = slices.DeleteFunc(s.pending, func(p *job) bool { return p == j })
		s.finishLocked(j, StatusCancelled, "")
		return nil
	}
	// The running fetch returns promptly and run reports the cancellation.
	j.cancel()
	return nil
}

// Subscribe returns the active downloads, oldest first, and a channel that
// receives every later state change. Taking both under one lock means no
// update falls between them. Call unsubscribe when done; after Drain the
// channel is closed and Subscribe returns an already closed channel.
func (s *DownloadService) Subscribe() (active []Download, updates <-chan Download, unsubscribe func()) {
	ch := make(chan Download, subscriberBuffer)

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		close(ch)
		return nil, ch, func() {}
	}

	for _, j := range s.jobs {
		active = append(active, j.state)
	}
	slices.SortFunc(active, func(a, b Download) int {
		return cmp.Compare(s.jobs[a.ID].seq, s.jobs[b.ID].seq)
	})

	s.subscribers[ch] = struct{}{}
	return active, ch, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if _, ok := s.subscribers[ch]; ok {
			delete(s.subscribers, ch)
			close(ch)
		}
	}
}

// publishLocked sends d to every subscriber without blocking. s.mu must be
// held.
func (s *DownloadService) publishLocked(d Download) {
	for sub := range s.subscribers {
		select {
		case sub <- d:
			continue
		default:
		}

		// Dropping progress updates when the subscriber's channel buffer is full
		// is an intentional design decision to prevent slow or blocked clients
		// from blocking downloads.
		if !d.Status.Terminal() {
			continue
		}

		// A terminal update must reach the client, otherwise its UI never leaves
		// the downloading state. Make room by discarding the oldest buffered
		// update, which is almost always a stale progress update.
		select {
		case <-sub:
		default:
		}
		select {
		case sub <- d:
		default:
			slog.Warn("dropped terminal download update", "id", d.ID, "status", d.Status)
		}
	}
}

// Drain stops accepting downloads and cancels queued ones, then waits for
// running downloads to finish. Downloads still running when ctx is done are
// cancelled. Once every download has ended, all subscriber channels are closed
// so that long-lived event streams end and the HTTP server can shut down.
func (s *DownloadService) Drain(ctx context.Context) {
	s.mu.Lock()
	s.draining = true
	queued := s.pending
	s.pending = nil
	for _, j := range queued {
		s.finishLocked(j, StatusCancelled, "")
	}
	s.mu.Unlock()

	done := make(chan struct{})
	go func() {
		s.inFlight.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		s.mu.Lock()
		slog.Warn("shutdown timeout reached, cancelling downloads", "count", len(s.jobs))
		for _, j := range s.jobs {
			j.cancel()
		}
		s.mu.Unlock()
		// Fetchers stop promptly once cancelled.
		<-done
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for sub := range s.subscribers {
		close(sub)
	}
	clear(s.subscribers)
	s.closed = true
}
