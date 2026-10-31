package download

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

// ErrDraining is returned by StartDownload once the manager is shutting down.
var ErrDraining = errors.New("server is shutting down")

// DownloadManager manages video downloads and subscriptions for progress updates.
type DownloadManager struct {
	downloader    Downloader
	subscribers   []chan ProgressUpdate
	cancellations map[string]context.CancelFunc
	inFlight      sync.WaitGroup
	draining      bool
	closed        bool
	mux           sync.RWMutex
}

// NewDownloadManager creates and returns a new DownloadManager.
func NewDownloadManager(downloader Downloader) *DownloadManager {
	return &DownloadManager{
		downloader:    downloader,
		subscribers:   make([]chan ProgressUpdate, 0),
		cancellations: make(map[string]context.CancelFunc),
	}
}

const (
	statusDownloading = "downloading"
	statusCancelled   = "cancelled"
	statusError       = "error"
	statusFinished    = "finished"
)

// StartDownload starts a new video download in a goroutine and returns a unique identifier.
func (m *DownloadManager) StartDownload(ctx context.Context, req *DownloadRequest) (string, error) {
	u := uuid.New()
	uStr := u.String()

	bgCtx, cancel := context.WithCancel(context.Background())

	m.mux.Lock()
	if m.draining {
		m.mux.Unlock()
		cancel()
		return "", ErrDraining
	}
	// Add under the lock so it cannot race with Drain's Wait.
	m.inFlight.Add(1)
	m.cancellations[uStr] = cancel
	m.mux.Unlock()

	go func() {
		defer func() {
			m.mux.Lock()
			delete(m.cancellations, uStr)
			m.mux.Unlock()
			cancel()
			m.inFlight.Done()
		}()

		m.broadcast(ProgressUpdate{
			UUID:    uStr,
			VideoID: req.VideoID,
			Title:   req.Title,
			Status:  statusDownloading,
		})

		err := m.downloader.Download(bgCtx, req, func(p ProgressUpdate) {
			p.UUID = uStr
			p.VideoID = req.VideoID
			p.Title = req.Title
			p.Status = statusDownloading
			m.broadcast(p)
		})

		if bgCtx.Err() != nil {
			m.broadcast(ProgressUpdate{
				UUID:    uStr,
				VideoID: req.VideoID,
				Title:   req.Title,
				Status:  statusCancelled,
			})
		} else if err != nil {
			m.broadcast(ProgressUpdate{
				UUID:         uStr,
				VideoID:      req.VideoID,
				Title:        req.Title,
				Status:       statusError,
				ErrorMessage: err.Error(),
			})
		} else {
			m.broadcast(ProgressUpdate{
				UUID:    uStr,
				VideoID: req.VideoID,
				Title:   req.Title,
				Status:  statusFinished,
				Percent: "100%",
			})
		}
	}()

	return uStr, nil
}

// CancelDownload cancels an active download by its UUID.
func (m *DownloadManager) CancelDownload(uuid string) bool {
	m.mux.Lock()
	cancel, ok := m.cancellations[uuid]
	m.mux.Unlock()
	if ok {
		cancel()
		return true
	}
	return false
}

// Drain stops accepting new downloads and waits for in-flight ones to finish.
// Downloads still running when ctx is done are cancelled. Once every download
// has ended, all subscriber channels are closed so that long-lived event
// streams terminate and the HTTP server can shut down.
func (m *DownloadManager) Drain(ctx context.Context) {
	m.mux.Lock()
	m.draining = true
	m.mux.Unlock()

	done := make(chan struct{})
	go func() {
		m.inFlight.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		m.mux.RLock()
		slog.Warn("shutdown timeout reached, cancelling downloads", "count", len(m.cancellations))
		for _, cancel := range m.cancellations {
			cancel()
		}
		m.mux.RUnlock()
		// Cancellation kills yt-dlp, so the remaining downloads return promptly.
		<-done
	}

	m.mux.Lock()
	defer m.mux.Unlock()
	for _, sub := range m.subscribers {
		close(sub)
	}
	m.subscribers = nil
	m.closed = true
}

// Subscribe returns a channel that receives progress updates for all active
// downloads. After Drain, it returns an already closed channel.
func (m *DownloadManager) Subscribe() chan ProgressUpdate {
	ch := make(chan ProgressUpdate, 10)
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.closed {
		close(ch)
		return ch
	}
	m.subscribers = append(m.subscribers, ch)
	return ch
}

// Unsubscribe removes a channel from the list of subscribers and closes it.
func (m *DownloadManager) Unsubscribe(ch chan ProgressUpdate) {
	m.mux.Lock()
	defer m.mux.Unlock()

	for i, sub := range m.subscribers {
		if sub == ch {
			m.subscribers = append(m.subscribers[:i], m.subscribers[i+1:]...)
			close(ch)
			break
		}
	}
}

func isTerminal(status string) bool {
	return status == statusCancelled || status == statusError || status == statusFinished
}

func (m *DownloadManager) broadcast(p ProgressUpdate) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	for _, sub := range m.subscribers {
		select {
		case sub <- p:
			continue
		default:
		}

		// Dropping progress updates when the subscriber's channel buffer is full
		// is an intentional design decision to prevent slow or blocked SSE clients
		// from blocking the main download goroutine.
		if !isTerminal(p.Status) {
			continue
		}

		// A terminal update must reach the client, otherwise its UI never leaves
		// the downloading state. Make room by discarding the oldest buffered
		// update, which is almost always a stale progress tick.
		select {
		case <-sub:
		default:
		}
		select {
		case sub <- p:
		default:
			slog.Warn("dropped terminal download update", "uuid", p.UUID, "status", p.Status)
		}
	}
}
