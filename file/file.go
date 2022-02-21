package file

import (
	"io/fs"
	"io/ioutil"
)

type File struct {
	fs.FileInfo
	DiskName string
}

func (this *File) Get() string {
	contents, _ := ioutil.ReadFile(this.Name())
	return string(contents)
}

func (this *File) Disk() string {
	return this.DiskName
}
