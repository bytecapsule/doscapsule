//go:build !windows

package cli

import (
	"os"
	"syscall"
)

// terminationSignals covers Ctrl+C, `kill` and closing the terminal. Catching
// SIGHUP means the foreground game is killed and reaped instead of being left
// behind when the terminal goes away.
func terminationSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP}
}
