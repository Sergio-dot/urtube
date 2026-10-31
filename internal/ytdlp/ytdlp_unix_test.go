//go:build unix

package ytdlp

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFetcher_CancelKillsChildren(t *testing.T) {
	f := &Fetcher{TempDir: t.TempDir()}
	job := testJob(audioMP3)
	job.Dir = t.TempDir()

	pidFile := filepath.Join(t.TempDir(), "child.pid")
	// The child stands in for ffmpeg: it would outlive yt-dlp if only
	// the direct process were killed.
	f.Executable = fakeExecutable(t, `
sleep 30 &
echo $! > `+pidFile+`
wait`)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- f.Fetch(ctx, job, nil) }()

	var childPID int
	assert.Eventually(t, func() bool {
		b, err := os.ReadFile(pidFile)
		childPID, _ = strconv.Atoi(strings.TrimSpace(string(b)))
		return err == nil && childPID > 0
	}, 2*time.Second, 10*time.Millisecond)

	cancel()
	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("Fetch did not return after cancellation")
	}

	assert.Eventually(t, func() bool {
		// Signal 0 checks for existence; ESRCH means the child is gone.
		return syscall.Kill(childPID, 0) == syscall.ESRCH
	}, 2*time.Second, 10*time.Millisecond, "child process %d survived cancellation", childPID)
}
