// Package settings keeps the settings users change from the web app, saved
// as JSON in the data directory so they survive restarts. Environment
// configuration provides the defaults.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Sergio-dot/urtube/internal/health"
	"github.com/Sergio-dot/urtube/internal/media"
)

// FileName is the settings file inside the data directory.
const FileName = "settings.json"

// saved is the on-disk format. Empty fields mean "use the default".
type saved struct {
	DownloadDir string `json:"downloadDir,omitempty"`
}

// Store holds the current settings and saves changes to its file. It
// implements media.Destination.
type Store struct {
	path               string
	defaultDownloadDir string
	root               string

	mu    sync.RWMutex
	saved saved
}

// Open loads the settings saved in dataDir, if any. defaultDownloadDir
// applies until the user picks another folder. If root is not empty, every
// download folder must be inside it. A missing file is not an error; an
// unreadable one is, but the returned Store still works with the defaults
// and overwrites the file on the next change.
func Open(dataDir, defaultDownloadDir, root string) (*Store, error) {
	s := &Store{
		path:               filepath.Join(dataDir, FileName),
		defaultDownloadDir: defaultDownloadDir,
		root:               root,
	}

	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, fmt.Errorf("read settings: %w", err)
	}
	if err := json.Unmarshal(data, &s.saved); err != nil {
		return s, fmt.Errorf("parse %s: %w", s.path, err)
	}
	return s, nil
}

// DownloadDir returns the folder downloads are saved to, and an error if it
// can't be used: outside the root (e.g. after the root changed) or not
// writable.
func (s *Store) DownloadDir() (string, error) {
	dir := s.CurrentDownloadDir()
	if err := s.checkInRoot(dir); err != nil {
		return dir, err
	}
	return dir, health.CheckWritableDir(dir)
}

// Root returns the folder download folders must be inside, or "" if any
// folder is allowed.
func (s *Store) Root() string {
	return s.root
}

func (s *Store) checkInRoot(dir string) error {
	if s.root == "" {
		return nil
	}
	rel, err := filepath.Rel(s.root, filepath.Clean(dir))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return media.InvalidInput(fmt.Sprintf("download folder must be inside %s", s.root))
	}
	return nil
}

// CurrentDownloadDir returns the folder downloads are saved to, without
// checking it.
func (s *Store) CurrentDownloadDir() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.saved.DownloadDir != "" {
		return s.saved.DownloadDir
	}
	return s.defaultDownloadDir
}

// DefaultDownloadDir returns the folder used when none has been chosen.
func (s *Store) DefaultDownloadDir() string {
	return s.defaultDownloadDir
}

// SetDownloadDir switches downloads to dir after checking that it is an
// absolute path to a writable folder (created if missing), and saves the
// choice. An empty dir goes back to the default. Downloads already started
// keep their folder.
func (s *Store) SetDownloadDir(dir string) error {
	if dir != "" {
		if !filepath.IsAbs(dir) {
			return media.InvalidInput("download folder must be an absolute path, such as /srv/media/music")
		}
		dir = filepath.Clean(dir)
		if err := s.checkInRoot(dir); err != nil {
			return err
		}
		if err := health.CheckWritableDir(dir); err != nil {
			return media.InvalidInput(err.Error())
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.saved
	next.DownloadDir = dir
	if err := s.writeLocked(next); err != nil {
		return err
	}
	s.saved = next
	return nil
}

// writeLocked saves v atomically, so a crash never leaves a half-written
// file. s.mu must be held.
func (s *Store) writeLocked(v saved) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	tmp, err := os.CreateTemp(dir, FileName+".*")
	if err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // no-op after a successful rename

	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close() // the write error is the one to report
		return fmt.Errorf("save settings: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	if err := os.Rename(tmp.Name(), s.path); err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	return nil
}
