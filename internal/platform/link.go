// Package platform contains small OS-specific safety primitives.
package platform

import "io/fs"

func IsLinkOrReparse(path string, info fs.FileInfo) bool {
	return info.Mode()&fs.ModeSymlink != 0 || isReparsePoint(path)
}
