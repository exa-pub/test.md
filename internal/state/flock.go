package state

import (
	"os"
	"syscall"
)

// Lock acquires an exclusive advisory lock on the given path.
// Creates the file if it doesn't exist.
// Returns the file handle — caller must pass it to Unlock when done.
func Lock(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

// Unlock releases the advisory lock and closes the file.
func Unlock(f *os.File) error {
	if f == nil {
		return nil
	}
	syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return f.Close()
}
