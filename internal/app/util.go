package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func launchConfDir() (string, error) {
	dir, err := stateDir()
	if err != nil {
		return "", err
	}
	p := filepath.Join(dir, "launch")
	if err := os.MkdirAll(p, 0o755); err != nil {
		return "", err
	}
	return p, nil
}

// resolveGameDir maps an id (a path relative to the root) back to its
// directory, refusing anything that would escape the root.
func resolveGameDir(root, id string) (string, error) {
	dir := filepath.Clean(filepath.Join(root, id))
	rootAbs, _ := filepath.Abs(root)
	dirAbs, _ := filepath.Abs(dir)
	if !strings.HasPrefix(dirAbs, rootAbs+string(os.PathSeparator)) || !isDir(dirAbs) {
		return "", fmt.Errorf("%w: %s", ErrGameNotFound, id)
	}
	return dirAbs, nil
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}
