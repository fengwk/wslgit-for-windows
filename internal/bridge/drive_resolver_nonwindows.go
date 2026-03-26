//go:build !windows

package bridge

var defaultDriveRemoteResolver DriveRemoteResolver = noopDriveRemoteResolver{}
