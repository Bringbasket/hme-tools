//go:build !windows

package storage

import "os"

func replaceFileAtomically(source, target string) error {
	return os.Rename(source, target)
}
