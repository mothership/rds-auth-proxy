package file

import "github.com/spf13/afero"

var appFs = afero.NewOsFs()

func GetFileSystem() afero.Fs {
	return appFs
}
