//go:build unix

package ytdlp

import (
	"os/exec"
	"syscall"
)

// killProcessGroupOnCancel runs cmd as the leader of a new process group and
// makes context cancellation kill the whole group. Killing only yt-dlp would
// leave its ffmpeg children running.
func killProcessGroupOnCancel(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
