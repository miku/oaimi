package next

import "testing"

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
