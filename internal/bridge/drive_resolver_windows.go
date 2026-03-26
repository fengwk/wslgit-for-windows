//go:build windows

package bridge

import (
	"fmt"
	"syscall"
	"unsafe"
)

var defaultDriveRemoteResolver DriveRemoteResolver = windowsDriveRemoteResolver{}

type windowsDriveRemoteResolver struct{}

const errorNotConnected syscall.Errno = 2250

var (
	mprDLL                = syscall.NewLazyDLL("mpr.dll")
	procWNetGetConnection = mprDLL.NewProc("WNetGetConnectionW")
)

func (windowsDriveRemoteResolver) Resolve(drive string) (string, bool, error) {
	localName, err := syscall.UTF16PtrFromString(drive)
	if err != nil {
		return "", false, err
	}

	buffer := make([]uint16, 512)
	bufferLength := uint32(len(buffer))
	r1, _, callErr := procWNetGetConnection.Call(
		uintptr(unsafe.Pointer(localName)),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&bufferLength)),
	)

	switch syscall.Errno(r1) {
	case 0:
		return syscall.UTF16ToString(buffer[:bufferLength]), true, nil
	case errorNotConnected:
		return "", false, nil
	case syscall.ERROR_MORE_DATA:
		buffer = make([]uint16, bufferLength)
		r1, _, callErr = procWNetGetConnection.Call(
			uintptr(unsafe.Pointer(localName)),
			uintptr(unsafe.Pointer(&buffer[0])),
			uintptr(unsafe.Pointer(&bufferLength)),
		)
		if syscall.Errno(r1) == 0 {
			return syscall.UTF16ToString(buffer[:bufferLength]), true, nil
		}
		if syscall.Errno(r1) == errorNotConnected {
			return "", false, nil
		}
	}

	if callErr != syscall.Errno(0) {
		return "", false, fmt.Errorf("解析映射盘 %s 失败: %w", drive, callErr)
	}

	return "", false, fmt.Errorf("解析映射盘 %s 失败: %v", drive, syscall.Errno(r1))
}
