package file

import (
	"bytes"
	"fmt"

	"github.com/spf13/afero"
)

type FileWriter struct {
	buffer *bytes.Buffer
	Fs     afero.Fs
	err    error
}

func NewFileWriter() *FileWriter {
	return &FileWriter{
		buffer: bytes.NewBuffer(make([]byte, 0, 120)),
		Fs:     appFs,
	}
}

func (f *FileWriter) P(fmtStr string, args ...interface{}) {
	if f.err != nil {
		return
	}
	_, err := f.buffer.WriteString(fmt.Sprintf(fmtStr, args...))
	if err != nil {
		f.err = err
		return
	}
	_, err = f.buffer.WriteString("\n")
	f.err = err
}

func (f *FileWriter) Write(bytes []byte) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	count, err := f.buffer.Write(bytes)
	f.err = err
	return count, err
}

func (f *FileWriter) Save(path string) error {
	if f.err != nil {
		return f.err
	}

	tmpfile, err := afero.TempFile(f.Fs, "", "file*")
	if err != nil {
		return err
	}
	// Note: We don't check errors here because a successful
	//       write means the tmpfile won't exist anymore
	//nolint:errcheck
	defer f.Fs.Remove(tmpfile.Name())
	if _, err := f.buffer.WriteTo(tmpfile); err != nil {
		return err
	}
	if err = tmpfile.Sync(); err != nil {
		return err
	}
	if err = tmpfile.Close(); err != nil {
		return err
	}

	return f.Fs.Rename(tmpfile.Name(), path)
}
