//go:build windows

package ssh

import "os"

// Windows lacks flock; the atomic temp-file+rename still protects integrity.
func lockFile(f *os.File) error   { return nil }
func unlockFile(f *os.File) error { return nil }
