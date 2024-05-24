package adapters

import (
	"io/fs"
	"os"
)

type File struct {
	fs.FileInfo
	DiskName string
	path     string
}

func (file *File) Read() []byte {
	var contents, _ = os.ReadFile(file.path)
	return contents
}

func (file *File) ReadString() string {
	contents, _ := os.ReadFile(file.path)
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

func (qiniuFile *QiniuFile) Disk() string {
	return qiniuFile.DiskName
}

func (qiniuFile *QiniuFile) Read() []byte {
	var bytes, _ = qiniuFile.disk.Read(qiniuFile.Name())
	return bytes
}

func (qiniuFile *QiniuFile) ReadString() string {
	var contents, _ = qiniuFile.disk.Get(qiniuFile.Name())
	return contents
}
