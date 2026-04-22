package update

import "os"

// isTerminal reports whether f is a character device (i.e. a real terminal).
// Used instead of pulling in golang.org/x/term — the posix-mode check is
// enough for the update-notice use case, where we only care about skipping
// the notice when output is piped/redirected/captured.
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
