// Package settings keeps the settings users change from the web app, saved
// as JSON in the data directory so they survive restarts. Environment
// configuration provides the defaults.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Sergio-dot/urtube/internal/health"
	"github.com/Sergio-dot/urtube/internal/media"
)

// FileName is the settings file inside the data directory.
const FileName = "settings.json"

// saved is the on-disk format. Empty fields mean "use the default". The
// file is only readable by its owner, as it can hold the Jellyfin API key.
type saved struct {
	DownloadDir    string `json:"downloadDir,omitempty"`
	JellyfinURL    string `json:"jellyfinUrl,omitempty"`
	JellyfinAPIKey string `json:"jellyfinApiKey,omitempty"`
}

// Defaults are the settings from the environment, used until the user
// changes them in the app.
type Defaults struct {
	DownloadDir string
	// DownloadRoot, if not empty, is the folder every download folder must
	// be inside.
	DownloadRoot   string
	JellyfinURL    string
	JellyfinAPIKey string
}

// Store holds the current settings and saves changes to its file. It
// implements media.Destination.
type Store struct {
	path               string
	defaultDownloadDir string
	root               string
	defaultJellyfinURL string
	defaultJellyfinKey string

	mu    sync.RWMutex
	saved saved
}

// Open loads the settings saved in dataDir, if any, on top of defaults. A
// missing file is not an error; an unreadable one is, but the returned
// Store still works with the defaults and overwrites the file on the next
// change.
func Open(dataDir string, defaults Defaults) (*Store, error) {
	s := &Store{
		path:               filepath.Join(dataDir, FileName),
		defaultDownloadDir: defaults.DownloadDir,
		root:               defaults.DownloadRoot,
		defaultJellyfinURL: defaults.JellyfinURL,
		defaultJellyfinKey: defaults.JellyfinAPIKey,
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

// Jellyfin returns the Jellyfin URL and API key; either may be empty, which
// means library refreshes are off.
func (s *Store) Jellyfin() (url, apiKey string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	url, apiKey = s.saved.JellyfinURL, s.saved.JellyfinAPIKey
	if url == "" {
		url = s.defaultJellyfinURL
	}
	if apiKey == "" {
		apiKey = s.defaultJellyfinKey
	}
	return url, apiKey
}

// SetJellyfin saves the Jellyfin URL and, unless apiKey is nil, the API
// key. Empty values go back to the defaults from the environment.
func (s *Store) SetJellyfin(url string, apiKey *string) error {
	url, err := ValidJellyfinURL(url)
	if err != nil {
		return err
	}
	if apiKey != nil {
		key := strings.TrimSpace(*apiKey)
		if len(key) > 200 || strings.ContainsFunc(key, func(r rune) bool { return r <= ' ' || r == '"' }) {
			return media.InvalidInput("the Jellyfin API key looks wrong: copy it again from Jellyfin's dashboard")
		}
		apiKey = &key
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.saved
	next.JellyfinURL = url
	if apiKey != nil {
		next.JellyfinAPIKey = *apiKey
	}
	if err := s.writeLocked(next); err != nil {
		return err
	}
	s.saved = next
	return nil
}

// ValidJellyfinURL checks that raw is empty or an absolute http(s) URL, and
// returns it without a trailing slash.
func ValidJellyfinURL(raw string) (string, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return "", nil
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.RawQuery != "" {
		return "", media.InvalidInput("the Jellyfin URL must look like http://192.168.1.10:8096")
	}
	return raw, nil
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
