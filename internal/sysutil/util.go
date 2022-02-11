//go:build !windows
// +build !windows

package sysutil

import "golang.org/x/sys/unix"

// RlimitStack reports the current stack size limit in bytes.
func RlimitStack() (cur uint64, err error) {
	var r unix.Rlimit
	err = unix.Getrlimit(unix.RLIMIT_STACK, &r)
	return r.Cur, err
}
