// Package ytdlp implements urtube's media.Fetcher and media.VideoSource with
// the yt-dlp command line tool. It is the only package that knows about
// yt-dlp.
package ytdlp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Sergio-dot/urtube/internal/media"
	"github.com/lrstanley/go-ytdlp"
)

const defaultExecutable = "yt-dlp"

// tempDirPattern names the per-download directories created in TempDir. The
// prefix is distinctive so RemoveStaleTempDirs only ever matches urtube's own.
const tempDirPattern = "urtube-download-*"

// progressPrefix marks the progress lines yt-dlp prints for --progress-template.
const progressPrefix = "urtube-progress:"

// passthroughEnv lists the parent environment variables forwarded to yt-dlp.
// Everything else (including any secrets in the container environment) is
// withheld. PATH is required for yt-dlp to find ffmpeg.
var passthroughEnv = []string{
	"PATH", "HOME", "TMPDIR", "XDG_CACHE_HOME", "LANG", "LC_ALL", "TZ",
	"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "no_proxy",
}

// Fetcher downloads videos with yt-dlp.
type Fetcher struct {
	// Executable is the name or path of yt-dlp. Empty uses "yt-dlp".
	Executable string
	// DownloadDir is the directory where finished downloads are saved.
	DownloadDir string
	// TempDir is the parent of the per-download directories that hold
	// in-progress files. Empty uses the system default.
	TempDir string
}

// Fetch downloads req with yt-dlp. In-progress and intermediate files are
// written to a temporary directory, removed afterwards, and only finished
// files are moved into DownloadDir.
func (f *Fetcher) Fetch(ctx context.Context, req media.Request, onProgress func(media.Progress)) error {
	tempDir, err := os.MkdirTemp(f.TempDir, tempDirPattern)
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	// Removes partial files left behind by a failed or cancelled download.
	defer os.RemoveAll(tempDir)

	// go-ytdlp only builds the command line: urtube runs the process itself so
	// that cancelling also kills yt-dlp's children (see killProcessGroupOnCancel).
	cmd := f.command(req.Options, tempDir).BuildCommand(ctx, req.URL)
	if cmd.Err != nil {
		return fmt.Errorf("resolve yt-dlp: %w", cmd.Err)
	}
	killProcessGroupOnCancel(cmd)

	cmd.Stdout = &lineWriter{onLine: func(line []byte) {
		if p, ok := parseProgress(line); ok && onProgress != nil {
			onProgress(p)
		}
	}}
	var lastError string
	cmd.Stderr = &lineWriter{onLine: func(line []byte) {
		if msg, ok := strings.CutPrefix(string(line), "ERROR: "); ok {
			lastError = msg
		}
	}}

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if lastError != "" {
			return errors.New(lastError)
		}
		return fmt.Errorf("yt-dlp: %w", err)
	}
	return nil
}

// RemoveStaleTempDirs deletes per-download directories left in TempDir by a
// previous run that was killed mid-download, such as a container stopped
// after its grace period. Call it at startup, before any download starts.
func (f *Fetcher) RemoveStaleTempDirs() error {
	if f.TempDir == "" {
		// The system temp dir is shared; never sweep it.
		return nil
	}
	dirs, err := filepath.Glob(filepath.Join(f.TempDir, tempDirPattern))
	if err != nil {
		return err
	}
	var errs []error
	for _, dir := range dirs {
		errs = append(errs, os.RemoveAll(dir))
	}
	return errors.Join(errs...)
}

// newCommand returns a yt-dlp command that never updates itself and only
// sees the passthroughEnv subset of the environment.
func newCommand(executable string) *ytdlp.Command {
	if executable == "" {
		executable = defaultExecutable
	}
	cmd := ytdlp.New().
		SetExecutable(executable).
		SetEnvVarInherit(false).
		NoUpdate()
	for _, k := range passthroughEnv {
		if v, ok := os.LookupEnv(k); ok {
			cmd.SetEnvVar(k, v)
		}
	}
	return cmd
}

// command builds the yt-dlp command line for opts.
func (f *Fetcher) command(opts media.Options, tempDir string) *ytdlp.Command {
	cmd := newCommand(f.Executable).
		Paths("home:" + f.DownloadDir).
		Paths("temp:" + tempDir).
		// Must stay relative: yt-dlp ignores --paths for an absolute output template.
		Output("%(title)s.%(ext)s").
		// One JSON progress line every 100ms on stdout, parsed by parseProgress.
		Progress().
		Newline().
		ProgressDelta(0.1).
		ProgressTemplate("download:" + progressPrefix + "%(progress)j")

	switch opts.Type {
	case media.MediaAudio:
		cmd.ExtractAudio().AudioFormat(opts.Format)
	case media.MediaVideo:
		cmd.Format("bestvideo+bestaudio/best").RecodeVideo(opts.Format)
	}

	return cmd
}

// progressLine is the subset of yt-dlp's progress dictionary urtube uses.
// yt-dlp prints null for unknown values, which decodes as zero.
type progressLine struct {
	DownloadedBytes    float64 `json:"downloaded_bytes"`
	TotalBytes         float64 `json:"total_bytes"`
	TotalBytesEstimate float64 `json:"total_bytes_estimate"`
	Speed              float64 `json:"speed"`
	ETA                float64 `json:"eta"`
}

// parseProgress decodes a progress line printed for --progress-template.
func parseProgress(line []byte) (media.Progress, bool) {
	raw, ok := bytes.CutPrefix(line, []byte(progressPrefix))
	if !ok {
		return media.Progress{}, false
	}
	var pl progressLine
	if err := json.Unmarshal(raw, &pl); err != nil {
		return media.Progress{}, false
	}

	total := pl.TotalBytes
	if total == 0 {
		total = pl.TotalBytesEstimate
	}
	return media.Progress{
		DownloadedBytes: int64(pl.DownloadedBytes),
		TotalBytes:      int64(total),
		BytesPerSecond:  pl.Speed,
		ETA:             time.Duration(pl.ETA * float64(time.Second)),
	}, true
}

// lineWriter calls onLine for every complete line written to it. yt-dlp
// rewrites console lines with carriage returns, so those end a line too.
type lineWriter struct {
	onLine func([]byte)
	buf    []byte
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexAny(w.buf, "\r\n")
		if i < 0 {
			return len(p), nil
		}
		if line := bytes.TrimSpace(w.buf[:i]); len(line) > 0 {
			w.onLine(line)
		}
		w.buf = w.buf[i+1:]
	}
}
