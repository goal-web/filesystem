package adapters

import (
	"bufio"
	"github.com/goal-web/contracts"
	"github.com/goal-web/supports/utils"
	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"io/fs"
	"time"
)

func QiniuAdapter(config contracts.Fields) contracts.FileSystem {
	return &Qiniu{
		name:   utils.GetStringField(config, "name"),
		bucket: utils.GetStringField(config, "bucket"),
		mac: qbox.NewMac(
			utils.GetStringField(config, "access_key"),
			utils.GetStringField(config, "secret_key"),
		),
	}
}

type Qiniu struct {
	name   string
	bucket string
	mac    *qbox.Mac
}

func (qiniu *Qiniu) Name() string {
	return qiniu.name
}

func (qiniu *Qiniu) Exists(path string) bool {
	return false
}

func (qiniu *Qiniu) Get(path string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (qiniu *Qiniu) ReadStream(path string) (*bufio.Reader, error) {
	//TODO implement me
	panic("implement me")
}

func (qiniu *Qiniu) Put(path, contents string) error {
	//TODO implement me
	panic("implement me")
}

func (qiniu *Qiniu) WriteStream(path string, contents string) error {
	//TODO implement me
	panic("implement me")
}

func (qiniu *Qiniu) GetVisibility(path string) contracts.FileVisibility {
	//TODO implement me
	panic("implement me")
}

func (qiniu *Qiniu) SetVisibility(path string, perm fs.FileMode) error {
	//TODO implement me
	panic("implement me")
}

func (qiniu *Qiniu) Prepend(path, contents string) error {
	//TODO implement me
	panic("implement me")
}

func (qiniu *Qiniu) Append(path, contents string) error {
	//TODO implement me
	panic("implement me")
}

func (qiniu *Qiniu) Delete(path string) error {
	//TODO implement me
	panic("implement me")
}

func (qiniu *Qiniu) Copy(from, to string) error {
	//TODO implement me
	panic("implement me")
}

func (qiniu *Qiniu) Move(from, to string) error {
	//TODO implement me
	panic("implement me")
}

func (qiniu *Qiniu) Size(path string) (int64, error) {
	//TODO implement me
	panic("implement me")
}

func (qiniu *Qiniu) LastModified(path string) (time.Time, error) {
	//TODO implement me
	panic("implement me")
}

func (qiniu *Qiniu) Files(directory string) []contracts.File {
	//TODO implement me
	panic("implement me")
}

func (qiniu *Qiniu) AllFiles(directory string) []contracts.File {
	//TODO implement me
	panic("implement me")
}

func (qiniu *Qiniu) Directories(directory string) []string {
	//TODO implement me
	panic("implement me")
}

func (qiniu *Qiniu) AllDirectories(directory string) []string {
	//TODO implement me
	panic("implement me")
}

func (qiniu *Qiniu) MakeDirectory(path string) error {
	//TODO implement me
	panic("implement me")
}

func (qiniu *Qiniu) DeleteDirectory(directory string) error {
	//TODO implement me
	panic("implement me")
}
