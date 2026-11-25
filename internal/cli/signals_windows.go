//go:build windows

package cli

import "os"

// Windows delivers Ctrl+C as os.Interrupt; SIGTERM/SIGHUP do not exist there.
func terminationSignals() []os.Signal { return []os.Signal{os.Interrupt} }
