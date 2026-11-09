package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Sergio-dot/urtube/internal/media"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore(t *testing.T) {
	t.Run("uses the default until a folder is chosen", func(t *testing.T) {
		def := t.TempDir()
		s, err := Open(t.TempDir(), Defaults{DownloadDir: def})
		require.NoError(t, err)

		dir, err := s.DownloadDir()
		assert.NoError(t, err)
		assert.Equal(t, def, dir)
		assert.Equal(t, def, s.DefaultDownloadDir())
	})

	t.Run("saves the chosen folder across restarts", func(t *testing.T) {
		dataDir, chosen := t.TempDir(), t.TempDir()
		s, err := Open(dataDir, Defaults{DownloadDir: "/default"})
		require.NoError(t, err)

		require.NoError(t, s.SetDownloadDir(chosen+"/"))

		reopened, err := Open(dataDir, Defaults{DownloadDir: "/default"})
		require.NoError(t, err)
		assert.Equal(t, chosen, reopened.CurrentDownloadDir(), "stored cleaned")
		entries, _ := os.ReadDir(dataDir)
		assert.Len(t, entries, 1, "no temp files left behind")
	})

	t.Run("empty folder restores the default", func(t *testing.T) {
		s, err := Open(t.TempDir(), Defaults{DownloadDir: "/default"})
		require.NoError(t, err)
		require.NoError(t, s.SetDownloadDir(t.TempDir()))

		require.NoError(t, s.SetDownloadDir(""))
		assert.Equal(t, "/default", s.CurrentDownloadDir())
	})

	t.Run("creates a missing folder", func(t *testing.T) {
		s, err := Open(t.TempDir(), Defaults{DownloadDir: "/default"})
		require.NoError(t, err)
		dir := filepath.Join(t.TempDir(), "music", "new")

		assert.NoError(t, s.SetDownloadDir(dir))
		assert.DirExists(t, dir)
	})

	t.Run("rejects relative paths", func(t *testing.T) {
		s, err := Open(t.TempDir(), Defaults{DownloadDir: "/default"})
		require.NoError(t, err)

		err = s.SetDownloadDir("music")
		assert.ErrorIs(t, err, media.ErrInvalidInput)
		assert.Equal(t, "/default", s.CurrentDownloadDir())
	})

	t.Run("rejects folders that aren't writable", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root bypasses file permissions")
		}
		s, err := Open(t.TempDir(), Defaults{DownloadDir: "/default"})
		require.NoError(t, err)
		dir := t.TempDir()
		require.NoError(t, os.Chmod(dir, 0o555))
		t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

		err = s.SetDownloadDir(dir)
		assert.ErrorIs(t, err, media.ErrInvalidInput)
		assert.EqualError(t, err, dir+" is not writable: permission denied")
		assert.Equal(t, "/default", s.CurrentDownloadDir())
	})

	t.Run("reports an unwritable current folder", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root bypasses file permissions")
		}
		def := t.TempDir()
		require.NoError(t, os.Chmod(def, 0o555))
		t.Cleanup(func() { _ = os.Chmod(def, 0o755) })
		s, err := Open(t.TempDir(), Defaults{DownloadDir: def})
		require.NoError(t, err)

		dir, err := s.DownloadDir()
		assert.Equal(t, def, dir)
		assert.ErrorContains(t, err, "not writable")
	})

	t.Run("keeps the old folder if saving fails", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root bypasses file permissions")
		}
		dataDir := t.TempDir()
		require.NoError(t, os.Chmod(dataDir, 0o555))
		t.Cleanup(func() { _ = os.Chmod(dataDir, 0o755) })
		s, err := Open(dataDir, Defaults{DownloadDir: "/default"})
		require.NoError(t, err)

		err = s.SetDownloadDir(t.TempDir())
		assert.ErrorContains(t, err, "save settings")
		assert.NotErrorIs(t, err, media.ErrInvalidInput)
		assert.Equal(t, "/default", s.CurrentDownloadDir())
	})

	t.Run("a corrupt file falls back to the defaults", func(t *testing.T) {
		dataDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dataDir, FileName), []byte("{nope"), 0o644))

		s, err := Open(dataDir, Defaults{DownloadDir: "/default"})
		assert.Error(t, err)
		require.NotNil(t, s)
		assert.Equal(t, "/default", s.CurrentDownloadDir())
		assert.NoError(t, s.SetDownloadDir(t.TempDir()), "the next change overwrites it")
	})

	t.Run("only allows folders inside the root", func(t *testing.T) {
		root := t.TempDir()
		s, err := Open(t.TempDir(), Defaults{DownloadDir: root, DownloadRoot: root})
		require.NoError(t, err)
		assert.Equal(t, root, s.Root())

		assert.NoError(t, s.SetDownloadDir(filepath.Join(root, "music")))
		assert.NoError(t, s.SetDownloadDir(root), "the root itself")

		for _, dir := range []string{
			filepath.Dir(root),
			filepath.Join(root, "..", "elsewhere"),
			root + "-sibling",
			"/tmp",
		} {
			err := s.SetDownloadDir(dir)
			assert.ErrorIs(t, err, media.ErrInvalidInput, dir)
			assert.EqualError(t, err, "download folder must be inside "+root, dir)
		}
		assert.Equal(t, root, s.CurrentDownloadDir())
	})

	t.Run("reports a saved folder outside a new root", func(t *testing.T) {
		dataDir, oldDir := t.TempDir(), t.TempDir()
		s, err := Open(dataDir, Defaults{DownloadDir: "/default"})
		require.NoError(t, err)
		require.NoError(t, s.SetDownloadDir(oldDir))

		root := t.TempDir()
		s, err = Open(dataDir, Defaults{DownloadDir: root, DownloadRoot: root})
		require.NoError(t, err)
		dir, err := s.DownloadDir()
		assert.Equal(t, oldDir, dir)
		assert.EqualError(t, err, "download folder must be inside "+root)
	})

	t.Run("Jellyfin defaults, saved values and clearing", func(t *testing.T) {
		dataDir := t.TempDir()
		s, err := Open(dataDir, Defaults{JellyfinURL: "http://env:8096", JellyfinAPIKey: "envkey"})
		require.NoError(t, err)
		url, key := s.Jellyfin()
		assert.Equal(t, "http://env:8096", url)
		assert.Equal(t, "envkey", key)

		key2 := " abc123 "
		require.NoError(t, s.SetJellyfin("http://jellyfin:8096/", &key2))
		reopened, err := Open(dataDir, Defaults{})
		require.NoError(t, err)
		url, key = reopened.Jellyfin()
		assert.Equal(t, "http://jellyfin:8096", url, "trailing slash dropped")
		assert.Equal(t, "abc123", key)

		require.NoError(t, reopened.SetJellyfin("http://jellyfin:8096/", nil))
		_, key = reopened.Jellyfin()
		assert.Equal(t, "abc123", key, "a nil key keeps the saved one")

		empty := ""
		require.NoError(t, reopened.SetJellyfin("", &empty))
		url, key = reopened.Jellyfin()
		assert.Empty(t, url)
		assert.Empty(t, key)
	})

	t.Run("a stored Jellyfin key stays with its server", func(t *testing.T) {
		empty, typed := "", "newkey"

		s, err := Open(t.TempDir(), Defaults{})
		require.NoError(t, err)
		saved := "abc123"
		require.NoError(t, s.SetJellyfin("http://jellyfin:8096", &saved))

		assert.ErrorIs(t, s.SetJellyfin("http://evil:8096", nil), media.ErrInvalidInput)
		url, key := s.Jellyfin()
		assert.Equal(t, "http://jellyfin:8096", url, "the rejected URL isn't saved")
		assert.Equal(t, "abc123", key)

		// Clearing the URL first must not open a way around the check.
		require.NoError(t, s.SetJellyfin("", nil))
		assert.ErrorIs(t, s.SetJellyfin("http://evil:8096", nil), media.ErrInvalidInput)

		require.NoError(t, s.SetJellyfin("http://other:8096", &typed), "a new key may go to a new server")
		url, key = s.Jellyfin()
		assert.Equal(t, "http://other:8096", url)
		assert.Equal(t, "newkey", key)

		env, err := Open(t.TempDir(), Defaults{JellyfinURL: "http://env:8096/", JellyfinAPIKey: "envkey"})
		require.NoError(t, err)
		assert.ErrorIs(t, env.SetJellyfin("http://evil:8096", nil), media.ErrInvalidInput, "the key from the environment too")
		assert.ErrorIs(t, env.SetJellyfin("http://evil:8096", &empty), media.ErrInvalidInput, "an empty key falls back to the environment's")
		require.NoError(t, env.SetJellyfin("http://env:8096", nil), "same server as the default")

		none, err := Open(t.TempDir(), Defaults{})
		require.NoError(t, err)
		require.NoError(t, none.SetJellyfin("http://first:8096", nil), "no key to protect yet")
		require.NoError(t, none.SetJellyfin("http://second:8096", nil))
	})

	t.Run("invalid Jellyfin settings", func(t *testing.T) {
		s, err := Open(t.TempDir(), Defaults{})
		require.NoError(t, err)
		for _, u := range []string{"jellyfin:8096", "ftp://x", "http://", "http://x/?a=b"} {
			assert.ErrorIs(t, s.SetJellyfin(u, nil), media.ErrInvalidInput, u)
		}
		bad := "two words"
		assert.ErrorIs(t, s.SetJellyfin("http://x", &bad), media.ErrInvalidInput)
	})

	t.Run("the settings file is private", func(t *testing.T) {
		dataDir := t.TempDir()
		s, err := Open(dataDir, Defaults{})
		require.NoError(t, err)
		key := "k"
		require.NoError(t, s.SetJellyfin("http://x", &key))
		info, err := os.Stat(filepath.Join(dataDir, FileName))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	})
}
