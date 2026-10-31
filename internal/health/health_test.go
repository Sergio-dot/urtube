package health

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckWritableDir(t *testing.T) {
	t.Run("creates a missing directory", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "nested", "downloads")

		assert.NoError(t, CheckWritableDir(dir))
		assert.DirExists(t, dir)

		entries, err := os.ReadDir(dir)
		assert.NoError(t, err)
		assert.Empty(t, entries, "the write check must clean up after itself")
	})

	t.Run("fails for a read-only directory", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root bypasses file permissions")
		}
		dir := t.TempDir()
		assert.NoError(t, os.Chmod(dir, 0o555))
		t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

		assert.ErrorContains(t, CheckWritableDir(dir), "not writable")
	})
}

func TestCheckDependencies(t *testing.T) {
	t.Run("fails when yt-dlp is not on PATH", func(t *testing.T) {
		t.Setenv("PATH", t.TempDir())

		_, err := CheckDependencies(context.Background())
		assert.ErrorContains(t, err, "yt-dlp not found")
	})

	t.Run("fails when ffmpeg is not on PATH", func(t *testing.T) {
		dir := t.TempDir()
		fakeExecutable(t, dir, "yt-dlp", "echo 2026.01.01")
		t.Setenv("PATH", dir)

		_, err := CheckDependencies(context.Background())
		assert.ErrorContains(t, err, "ffmpeg not found")
	})

	t.Run("returns the yt-dlp version", func(t *testing.T) {
		dir := t.TempDir()
		fakeExecutable(t, dir, "yt-dlp", "echo 2026.01.01")
		fakeExecutable(t, dir, "ffmpeg", "true")
		t.Setenv("PATH", dir)

		deps, err := CheckDependencies(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, "2026.01.01", deps.YtdlpVersion)
	})
}

// fakeExecutable writes a shell script named name into dir that runs body.
func fakeExecutable(t *testing.T, dir, name, body string) {
	t.Helper()
	script := "#!/bin/sh\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}
