package download

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// DownloadManager manages video downloads and subscriptions for progress updates.
type DownloadManager struct {
	downloader    Downloader
	subscribers   []chan ProgressUpdate
	cancellations map[string]context.CancelFunc
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
	m.cancellations[uStr] = cancel
	m.mux.Unlock()

	go func() {
		defer func() {
			m.mux.Lock()
			delete(m.cancellations, uStr)
			m.mux.Unlock()
			cancel()
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

// Subscribe returns a channel that receives progress updates for all active downloads.
func (m *DownloadManager) Subscribe() chan ProgressUpdate {
	ch := make(chan ProgressUpdate, 10)
	m.mux.Lock()
	defer m.mux.Unlock()
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

func (m *DownloadManager) broadcast(p ProgressUpdate) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	for _, sub := range m.subscribers {
		select {
		case sub <- p:
		default:
			// Dropping progress updates when the subscriber's channel buffer is full
			// is an intentional design decision to prevent slow or blocked SSE clients
			// from blocking the main download goroutine.
		}
	}
}
