package filesystem

import "os"

// Write atomically replaces the file at path with data: it writes a sibling
// temp file and renames it into place, so a crash never leaves a half-written
// item. It is the filesystem backend's persist step — the write dual of the
// read in this package, and what `fix` calls once it has computed the new bytes.
func Write(path string, data []byte) error {
	tmp, err := os.CreateTemp(dirOf(path), ".katalyst-fix-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// dirOf returns the directory of path, defaulting to "." when path has no
// separator. Used to keep the atomic temp file on the same filesystem.
func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}
