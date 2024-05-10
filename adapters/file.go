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

func (file *File) Read() []byte {
	var contents, _ = ioutil.ReadFile(file.path)
	return contents
}

func (file *File) ReadString() string {
	contents, _ := ioutil.ReadFile(file.path)
	return string(contents)
}

func (file *File) Disk() string {
	return file.DiskName
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
