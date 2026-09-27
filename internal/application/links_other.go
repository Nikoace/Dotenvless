//go:build !windows

package application

import "os"

func directoryLink(info os.FileInfo) bool { return info.Mode()&os.ModeSymlink != 0 }
