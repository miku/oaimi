package next

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var ErrBadKey = errors.New("bad key")

type Cache interface {
	Set(k, v string) error
	Get(k string) (string, error)
}

// DirCache caches values under a root directory. The key must be a valid
// relative path. A POSIX fully portable filename [A–Za–z0–9._-], hyphen must
// not be first character, filename length limit of OS.
type DirCache struct {
	directory string
}

func NewDirCache(directory string) (c DirCache, err error) {
	abs, err := filepath.Abs(directory)
	if err != nil {
		return c, err
	}
	return DirCache{abs}, err
}

func (c DirCache) cleanKey(k string) (s string, err error) {
	s = filepath.Clean(path.Join(c.directory, k))
	if !strings.HasPrefix(s, c.directory) {
		return "", ErrBadKey
	}
	return s, nil
}

func (c DirCache) Set(k, v string) error {
	pth, err := c.cleanKey(k)
	if err != nil {
		return err
	}
	dir := path.Dir(pth)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return WriteFileAtomic(pth, []byte(v), 0644)
}
