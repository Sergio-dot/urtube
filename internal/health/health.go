// Package health verifies at startup that urtube's external dependencies are
// usable, so a misconfigured container fails fast instead of serving errors.
package health

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// versionTimeout bounds how long "yt-dlp --version" may take.
const versionTimeout = 10 * time.Second

// Dependencies describes the external tools found on the host.
type Dependencies struct {
	// YtdlpVersion is the version reported by the yt-dlp executable.
	YtdlpVersion string
}

// CheckDependencies verifies that yt-dlp and ffmpeg are on PATH and returns the
// yt-dlp version. ffmpeg is required for audio extraction and video recoding.
func CheckDependencies(ctx context.Context) (Dependencies, error) {
	ytdlp, err := exec.LookPath("yt-dlp")
	if err != nil {
		return Dependencies{}, fmt.Errorf("yt-dlp not found: %w", err)
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return Dependencies{}, fmt.Errorf("ffmpeg not found: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, versionTimeout)
	defer cancel()

	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, ytdlp, "--version")
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return Dependencies{}, fmt.Errorf("run yt-dlp --version: %w", err)
	}

	return Dependencies{YtdlpVersion: strings.TrimSpace(out.String())}, nil
}

// CheckWritableDir creates dir if needed and verifies that files can be created
// in it, catching wrong ownership or a missing mount before any download runs.
func CheckWritableDir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("cannot create %s: %w", dir, pathCause(err))
	}

	f, err := os.CreateTemp(dir, ".urtube-write-check-*")
	if err != nil {
		return fmt.Errorf("%s is not writable: %w", dir, pathCause(err))
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		return fmt.Errorf("close %s: %w", name, err)
	}
	if err := os.Remove(name); err != nil {
		return fmt.Errorf("remove %s: %w", name, err)
	}
	return nil
}

// pathCause strips the path from a filesystem error, which already names it
// or names a scratch file the user never sees: "permission denied" rather
// than "open /x/.urtube-write-check-123: permission denied".
func pathCause(err error) error {
	var pe *os.PathError
	if errors.As(err, &pe) {
		return pe.Err
	}
	return err
}
