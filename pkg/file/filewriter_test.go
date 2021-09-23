package file

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/afero"
)

func TestFileWriter_Valid(t *testing.T) {
	cases := []struct {
		FileName string
		Content  []byte
	}{
		//Basic file in working directory
		{
			FileName: "/test.txt",
			Content:  []byte("hello world"),
		},
	}

	for idx, test := range cases {
		writer := NewFileWriter()
		writer.Fs = afero.NewMemMapFs()
		_, _ = writer.Write(test.Content)
		if err := writer.Save(test.FileName); err != nil {
			t.Errorf("[Case %d]: Error occured while writing file: %t", idx, err)
		}
		info, err := writer.Fs.Stat(test.FileName)
		if os.IsNotExist(err) || info.IsDir() {
			t.Errorf("[Case %d]: Could not locate the file FileWriter is expected to write to '%s'", idx, test.FileName)
		}
		fileBytes, err := afero.ReadFile(writer.Fs, test.FileName)
		if os.IsNotExist(err) || !bytes.Equal(fileBytes, test.Content) {
			t.Errorf("[Case %d]: Could not read the file FileWriter is expected to write to '%s'", idx, test.FileName)
		}
	}
}

func TestFileWriter_Invalid(t *testing.T) {
	cases := []struct {
		FileName      string
		Content       []byte
		ExpectedError string
	}{
		//attempt to write to home directory in a folder that does not exist
		{
			FileName:      "/badDir/test.txt",
			Content:       []byte("hello world"),
			ExpectedError: "no such file or directory",
		},
	}
	for idx, test := range cases {
		writer := NewFileWriter()
		testPath, _ := afero.TempDir(writer.Fs, "testDir", "testPrefix")
		_, _ = writer.Write(test.Content)
		err := writer.Save(testPath + test.FileName)
		if !strings.Contains(err.Error(), test.ExpectedError) {
			t.Errorf("[Case %d]: Unexpected error occured while writing file: %t", idx, err)
		}
	}
}
