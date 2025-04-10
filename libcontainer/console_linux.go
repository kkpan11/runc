package libcontainer

import (
	"os"

	"github.com/opencontainers/runc/internal/linux"
	"golang.org/x/sys/unix"
)

// mount initializes the console inside the rootfs mounting with the specified mount label
// and applying the correct ownership of the console.
func mountConsole(slavePath string) error {
	f, err := os.Create("/dev/console")
	if err != nil && !os.IsExist(err) {
		return err
	}
	if f != nil {
		// Ensure permission bits (can be different because of umask).
		if err := f.Chmod(0o666); err != nil {
			return err
		}
		f.Close()
	}
	return mount(slavePath, "/dev/console", "bind", unix.MS_BIND, "")
}

// dupStdio opens the slavePath for the console and dups the fds to the current
// processes stdio, fd 0,1,2.
func dupStdio(slavePath string) error {
	fd, err := linux.Open(slavePath, unix.O_RDWR, 0)
	if err != nil {
		return err
	}
	for _, i := range []int{0, 1, 2} {
		if err := linux.Dup3(fd, i, 0); err != nil {
			return err
		}
	}
	return nil
}
