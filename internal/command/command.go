package command

import (
	"os/exec"
	"runtime"

	"golang.org/x/sys/unix"
)

// Run runs an OS command.  On Linux it waits for the command to
// complete and returns the exit status (return code).
func Run(name string, arg ...string) (string, int, error) {
	cmd := exec.Command(name, arg...)
	// Run the command and wait for completion
	out, err := cmd.CombinedOutput()
	if err != nil {
		var rc int
		// Only works on Linux
		if runtime.GOOS == "linux" {
			var ws unix.WaitStatus
			unix.Wait4(cmd.Process.Pid, &ws, 0, nil)
			rc = ws.ExitStatus()
		} else {
			rc = -1
		}
		if rc == 0 {
			return string(out), rc, nil
		}
		return string(out), rc, err
	}
	return string(out), 0, nil
}
