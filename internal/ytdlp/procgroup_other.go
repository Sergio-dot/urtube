//go:build !unix

package ytdlp

import "os/exec"

// killProcessGroupOnCancel keeps exec's default of killing only yt-dlp on
// platforms without process groups.
func killProcessGroupOnCancel(cmd *exec.Cmd) {}
