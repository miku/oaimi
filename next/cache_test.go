package next

import "testing"

func TestNewDirCache(t *testing.T) {
	var tests = []struct {
		s   string
		dir string
		err error
	}{
		{"/", "/", nil},
		{"/hello/world/", "/hello/world", nil},
		{"/hello///world/", "/hello/world", nil},
		{"/hello/world", "/hello/world", nil},
	}
	for _, test := range tests {
		c, err := NewDirCache(test.s)
		if err != test.err {
			t.Errorf("NewDirCache(), got %v, want %v", err, test.err)
		}
		if c.directory != test.dir {
			t.Errorf("DirCache, wrong directory, got %v, want %v", c.directory, test.dir)
		}
	}
}

func TestCleanKey(t *testing.T) {
	var tests = []struct {
		key   string
		clean string
		err   error
	}{
		{"", "/tmp", nil},
		{"a", "/tmp/a", nil},
		{"abc", "/tmp/abc", nil},
		{"abc123", "/tmp/abc123", nil},
		{"-abc123", "/tmp/-abc123", nil},
		{"abc123/1/2/3", "/tmp/abc123/1/2/3", nil},
		{"../abc123/1/2/3", "", ErrBadKey},
		{"./../abc123/1/2/3", "", ErrBadKey},
		{"./../tmp/abc123/1/2/3", "/tmp/abc123/1/2/3", nil},
	}

	cache := DirCache{"/tmp"}
	for _, test := range tests {
		k, err := cache.cleanKey(test.key)
		if err != test.err {
			t.Errorf("c.cleanKey() got %v, want %v", err, test.err)
		}
		if k != test.clean {
			t.Errorf("c.cleanKey() got %v, want %v", k, test.clean)
		}
	}
}
