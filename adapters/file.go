package adapters

import (
	"io/fs"
	"io/ioutil"
)

type File struct {
	fs.FileInfo
	DiskName string
	path     string
}

func (this *File) Read() []byte {
	var contents, _ = ioutil.ReadFile(this.path)
	return contents
}

func (this *File) ReadString() string {
	contents, _ := ioutil.ReadFile(this.path)
	return string(contents)
}

func (this *File) Disk() string {
	return this.DiskName
}

type QiniuFile struct {
	QiniuFileInfo
	DiskName string
	disk     *Qiniu
}

func (this *QiniuFile) Disk() string {
	return this.DiskName
}

func (this *QiniuFile) Read() []byte {
	var bytes, _ = this.disk.Read(this.Name())
	return bytes
}

func (this *QiniuFile) ReadString() string {
	var contents, _ = this.disk.Get(this.Name())
	return contents
}
