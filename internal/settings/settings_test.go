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
		s, err := Open(t.TempDir(), def, "")
		require.NoError(t, err)

		dir, err := s.DownloadDir()
		assert.NoError(t, err)
		assert.Equal(t, def, dir)
		assert.Equal(t, def, s.DefaultDownloadDir())
	})

	t.Run("saves the chosen folder across restarts", func(t *testing.T) {
		dataDir, chosen := t.TempDir(), t.TempDir()
		s, err := Open(dataDir, "/default", "")
		require.NoError(t, err)

		require.NoError(t, s.SetDownloadDir(chosen+"/"))

		reopened, err := Open(dataDir, "/default", "")
		require.NoError(t, err)
		assert.Equal(t, chosen, reopened.CurrentDownloadDir(), "stored cleaned")
		entries, _ := os.ReadDir(dataDir)
		assert.Len(t, entries, 1, "no temp files left behind")
	})

	t.Run("empty folder restores the default", func(t *testing.T) {
		s, err := Open(t.TempDir(), "/default", "")
		require.NoError(t, err)
		require.NoError(t, s.SetDownloadDir(t.TempDir()))

		require.NoError(t, s.SetDownloadDir(""))
		assert.Equal(t, "/default", s.CurrentDownloadDir())
	})

	t.Run("creates a missing folder", func(t *testing.T) {
		s, err := Open(t.TempDir(), "/default", "")
		require.NoError(t, err)
		dir := filepath.Join(t.TempDir(), "music", "new")

		assert.NoError(t, s.SetDownloadDir(dir))
		assert.DirExists(t, dir)
	})

	t.Run("rejects relative paths", func(t *testing.T) {
		s, err := Open(t.TempDir(), "/default", "")
		require.NoError(t, err)

		err = s.SetDownloadDir("music")
		assert.ErrorIs(t, err, media.ErrInvalidInput)
		assert.Equal(t, "/default", s.CurrentDownloadDir())
	})

	t.Run("rejects folders that aren't writable", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root bypasses file permissions")
		}
		s, err := Open(t.TempDir(), "/default", "")
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
		s, err := Open(t.TempDir(), def, "")
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
		s, err := Open(dataDir, "/default", "")
		require.NoError(t, err)

		err = s.SetDownloadDir(t.TempDir())
		assert.ErrorContains(t, err, "save settings")
		assert.NotErrorIs(t, err, media.ErrInvalidInput)
		assert.Equal(t, "/default", s.CurrentDownloadDir())
	})

	t.Run("a corrupt file falls back to the defaults", func(t *testing.T) {
		dataDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dataDir, FileName), []byte("{nope"), 0o644))

		s, err := Open(dataDir, "/default", "")
		assert.Error(t, err)
		require.NotNil(t, s)
		assert.Equal(t, "/default", s.CurrentDownloadDir())
		assert.NoError(t, s.SetDownloadDir(t.TempDir()), "the next change overwrites it")
	})

	t.Run("only allows folders inside the root", func(t *testing.T) {
		root := t.TempDir()
		s, err := Open(t.TempDir(), root, root)
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
		s, err := Open(dataDir, "/default", "")
		require.NoError(t, err)
		require.NoError(t, s.SetDownloadDir(oldDir))

		root := t.TempDir()
		s, err = Open(dataDir, root, root)
		require.NoError(t, err)
		dir, err := s.DownloadDir()
		assert.Equal(t, oldDir, dir)
		assert.EqualError(t, err, "download folder must be inside "+root)
	})
}
