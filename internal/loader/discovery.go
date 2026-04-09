package loader

import (
	"os"
	"path/filepath"
	"strings"
)

// Discover walks dir recursively and returns absolute paths to all files whose
// names end in ".test.yaml" or ".test.yml".
//
// Directories whose names start with "." (hidden) or equal "node_modules" are
// skipped entirely so that vendor trees and dot-folders are never scanned.
func Discover(dir string) ([]string, error) {
	var paths []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			name := info.Name()
			// Skip the root dir itself; only prune children.
			if path != dir && (strings.HasPrefix(name, ".") || name == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}

		name := info.Name()
		if strings.HasSuffix(name, ".test.yaml") || strings.HasSuffix(name, ".test.yml") {
			abs, absErr := filepath.Abs(path)
			if absErr != nil {
				return absErr
			}
			paths = append(paths, abs)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}
