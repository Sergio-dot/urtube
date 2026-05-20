package download

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type DownloadManager struct {
	downloader  Downloader
	subscribers []chan ProgressUpdate
	mux         sync.RWMutex
}

func NewDownloadManager(downloader Downloader) *DownloadManager {
	return &DownloadManager{
		downloader:  downloader,
		subscribers: make([]chan ProgressUpdate, 0),
	}
}

func (m *DownloadManager) StartDownload(ctx context.Context, req *DownloadRequest) (string, error) {
	u := uuid.New()

	go func() {
		// Use Background context because the request context (ctx) will be canceled
		// as soon as the HTTP handler returns.
		bgCtx := context.Background()

		m.broadcast(ProgressUpdate{UUID: u.String(), Status: "downloading"})

		err := m.downloader.Download(bgCtx, req, func(p ProgressUpdate) {
			p.UUID = u.String()
			p.Status = "downloading"
			m.broadcast(p)
		})


		if err != nil {
			m.broadcast(ProgressUpdate{
				UUID:         u.String(),
				Status:       "error",
				ErrorMessage: err.Error(),
			})
		} else {
			m.broadcast(ProgressUpdate{UUID: u.String(), Status: "finished"})
		}
	}()

	return u.String(), nil
}

func (m *DownloadManager) Subscribe() chan ProgressUpdate {
	ch := make(chan ProgressUpdate, 10)
	m.mux.Lock()
	defer m.mux.Unlock()
	m.subscribers = append(m.subscribers, ch)
	return ch
}

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
		}
	}
}
