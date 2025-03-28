package devices

import (
	"fmt"

	"github.com/moby/sys/userns"
	"golang.org/x/sys/unix"

	"github.com/opencontainers/cgroups"
	devices "github.com/opencontainers/cgroups/devices/config"
)

func isRWM(perms devices.Permissions) bool {
	var r, w, m bool
	for _, perm := range perms {
		switch perm {
		case 'r':
			r = true
		case 'w':
			w = true
		case 'm':
			m = true
		}
	}
	return r && w && m
}

// This is similar to the logic applied in crun for handling errors from bpf(2)
// <https://github.com/containers/crun/blob/0.17/src/libcrun/cgroup.c#L2438-L2470>.
func canSkipEBPFError(r *cgroups.Resources) bool {
	// If we're running in a user namespace we can ignore eBPF rules because we
	// usually cannot use bpf(2), as well as rootless containers usually don't
	// have the necessary privileges to mknod(2) device inodes or access
	// host-level instances (though ideally we would be blocking device access
	// for rootless containers anyway).
	if userns.RunningInUserNS() {
		return true
	}

	// We cannot ignore an eBPF load error if any rule if is a block rule or it
	// doesn't permit all access modes.
	//
	// NOTE: This will sometimes trigger in cases where access modes are split
	//       between different rules but to handle this correctly would require
	//       using ".../libcontainer/cgroup/devices".Emulator.
	for _, dev := range r.Devices {
		if !dev.Allow || !isRWM(dev.Permissions) {
			return false
		}
	}
	return true
}

func setV2(dirPath string, r *cgroups.Resources) error {
	if r.SkipDevices {
		return nil
	}
	insts, license, err := deviceFilter(r.Devices)
	if err != nil {
		return err
	}
	dirFD, err := unix.Open(dirPath, unix.O_DIRECTORY|unix.O_RDONLY, 0o600)
	if err != nil {
		return fmt.Errorf("cannot get dir FD for %s", dirPath)
	}
	defer unix.Close(dirFD)
	if _, err := loadAttachCgroupDeviceFilter(insts, license, dirFD); err != nil {
		if !canSkipEBPFError(r) {
			return err
		}
	}
	return nil
}
