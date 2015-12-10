package oaimi

import (
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
)

const CompressThreshold = 1024

var (
	ErrFileNotWriteable = errors.New("not opened for writing")
	ErrFileNotReadable  = errors.New("not opened for reading")
)

type MaybeCompressedFile struct {
	w *compresswriter
	r *compressreader
}

// CreateMaybeCompressedFile creates a file, that may be compressed, if a
// certain amount of data is written to it.
func CreateMaybeCompressedFile(filename string) *MaybeCompressedFile {
	return &MaybeCompressedFile{w: &compresswriter{filename: filename}}
}

// OpenMaybeCompressedFile returns a file, that may be transparently
// decompressed on the fly.
func OpenMaybeCompressedFile(filename string) (*MaybeCompressedFile, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	var reader io.Reader
	gz, err := gzip.NewReader(bufio.NewReader(file))
	switch err {
	case nil:
		reader = gz
	case gzip.ErrHeader, io.ErrUnexpectedEOF, io.EOF:
		if _, err := file.Seek(0, os.SEEK_SET); err != nil {
			return nil, err
		}
		reader = bufio.NewReader(file)
	default:
		return nil, err
	}
	return &MaybeCompressedFile{r: &compressreader{r: reader, gz: gz, file: file}}, nil
}

func (f *MaybeCompressedFile) Name() string {
	if f.w != nil {
		return f.w.filename
	}
	if f.r != nil {
		return f.r.file.Name()
	}
	return ""
}

func (f *MaybeCompressedFile) Read(p []byte) (n int, err error) {
	if f.r == nil {
		return 0, ErrFileNotReadable
	}
	return f.r.Read(p)
}

func (f *MaybeCompressedFile) Write(p []byte) (n int, err error) {
	if f.w == nil {
		return 0, ErrFileNotWriteable
	}
	return f.w.Write(p)
}

func (f *MaybeCompressedFile) Close() error {
	if f.r != nil {
		return f.r.Close()
	}
	if f.w != nil {
		return f.w.Close()
	}
	return nil
}

// mkdirAll ensures a path exists and is a directory.
func mkdirAll(dir string) error {
	fi, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	if !fi.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}
	return nil
}

// compresswriter optionally compresses everything that is written to it.
type compresswriter struct {
	filename string
	tempfile *os.File
	bw       *bufio.Writer
	written  int
}

// init initializes internal fields
func (w *compresswriter) init() error {
	tf, err := ioutil.TempFile("", "compresswriter-")
	if err != nil {
		return err
	}
	w.tempfile = tf
	w.bw = bufio.NewWriter(tf)
	return nil
}

func (w *compresswriter) Write(p []byte) (n int, err error) {
	if w.tempfile == nil {
		if err := w.init(); err != nil {
			return 0, err
		}
	}
	w.written += len(p)
	return w.bw.Write(p)
}

func (w *compresswriter) Close() error {
	if w.tempfile == nil {
		if err := w.init(); err != nil {
			return err
		}
	}
	if err := w.bw.Flush(); err != nil {
		return err
	}
	if _, err := w.tempfile.Seek(0, os.SEEK_SET); err != nil {
		return err
	}

	// cleanup temporary file
	defer func() error {
		if err := w.tempfile.Close(); err != nil {
			return err
		}
		if err := os.Remove(w.tempfile.Name()); err != nil {
			return err
		}
		return nil
	}()

	if err := mkdirAll(path.Dir(w.filename)); err != nil {
		return err
	}

	if w.written < CompressThreshold {
		b, err := ioutil.ReadAll(w.tempfile)
		if err != nil {
			return err
		}
		if err := WriteFileAtomic(w.filename, b, 0644); err != nil {
			return err
		}
	} else {
		dir, name := path.Split(w.filename)
		file, err := ioutil.TempFile(dir, name)
		if err != nil {
			return err
		}
		gz := gzip.NewWriter(file)
		if _, err := io.Copy(gz, bufio.NewReader(w.tempfile)); err != nil {
			return err
		}
		if err := gz.Close(); err != nil {
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}
		return os.Rename(file.Name(), w.filename)
	}
	return nil
}

type compressreader struct {
	file *os.File
	r    io.Reader
	gz   *gzip.Reader
}

func (r *compressreader) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}

func (r *compressreader) Close() error {
	if r.gz != nil {
		if err := r.gz.Close(); err != nil {
			return err
		}
	}
	return r.file.Close()
}
