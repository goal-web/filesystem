package adapters

import (
	"bufio"
	"context"
	"github.com/goal-web/contracts"
	"github.com/goal-web/filesystem/file"
	"github.com/goal-web/supports/logs"
	"github.com/goal-web/supports/utils"
	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"
)

type QiniuFileInfo struct {
	isDir bool
	*storage.FileInfo
	*storage.ListItem
	name string
}

func (q QiniuFileInfo) Name() string {
	return q.name
}

func (q QiniuFileInfo) Size() int64 {
	if q.ListItem != nil {
		return q.ListItem.Fsize
	}
	return q.FileInfo.Fsize
}

func (q QiniuFileInfo) Mode() fs.FileMode {
	return os.ModePerm
}

func (q QiniuFileInfo) ModTime() time.Time {
	if q.ListItem != nil {
		return storage.ParsePutTime(q.ListItem.PutTime)
	}
	return storage.ParsePutTime(q.FileInfo.PutTime)
}

func (q QiniuFileInfo) IsDir() bool {
	return q.isDir
}

func (q QiniuFileInfo) Sys() interface{} {
	return nil
}

func QiniuAdapter(config contracts.Fields) contracts.FileSystem {
	var (
		mac = qbox.NewMac(
			utils.GetStringField(config, "access_key"),
			utils.GetStringField(config, "secret_key"),
		)
		bucketConfig, _ = config["config"].(*storage.Config)
	)
	return &Qiniu{
		name:          utils.GetStringField(config, "name"),
		domain:        utils.GetStringField(config, "domain"),
		private:       utils.GetBoolField(config, "private"),
		bucket:        utils.GetStringField(config, "bucket"),
		mac:           mac,
		bucketConfig:  bucketConfig,
		ttl:           time.Duration(utils.GetIntField(config, "ttl")) * time.Second,
		bucketManager: storage.NewBucketManager(mac, bucketConfig),
	}
}

type Qiniu struct {
	bucketConfig  *storage.Config
	name          string
	domain        string
	ttl           time.Duration
	private       bool
	bucket        string
	mac           *qbox.Mac
	bucketManager *storage.BucketManager
}

func (qiniu *Qiniu) Name() string {
	return qiniu.name
}

func (qiniu *Qiniu) Exists(path string) bool {
	var _, err = qiniu.bucketManager.Stat(qiniu.bucket, path)
	if err != nil {
		return false
	}
	return true
}

func (qiniu *Qiniu) Get(path string) (string, error) {
	var (
		url   string
		bytes []byte
	)
	if qiniu.private {
		url = storage.MakePrivateURL(qiniu.mac, qiniu.domain, path, time.Now().Add(qiniu.ttl).Unix())
	} else {
		url = storage.MakePublicURL(qiniu.domain, path)
	}

	var res, err = http.Get(url)
	if err != nil {
		return "", err
	}

	_, err = res.Body.Read(bytes)

	return string(bytes), err
}

func (qiniu *Qiniu) ReadStream(path string) (*bufio.Reader, error) {
	var (
		url string
	)
	if qiniu.private {
		url = storage.MakePrivateURL(qiniu.mac, qiniu.domain, path, time.Now().Add(qiniu.ttl).Unix())
	} else {
		url = storage.MakePublicURL(qiniu.domain, path)
	}

	var res, err = http.Get(url)
	if err != nil {
		return nil, err
	}

	return bufio.NewReader(res.Body), err
}

func (qiniu *Qiniu) Put(path, contents string) error {
	var (
		putPolicy = storage.PutPolicy{Scope: qiniu.bucket + ":" + path}
		token     = putPolicy.UploadToken(qiniu.mac)
	)

	var (
		uploader = storage.NewFormUploader(qiniu.bucketConfig)
		ret      = storage.PutRet{}
		reader   = strings.NewReader(contents)
		extra    = &storage.PutExtra{
			Params: map[string]string{
				//"x:name": "github logo",
			},
		}
	)

	return uploader.Put(context.Background(), &ret, token, path, reader, reader.Size(), extra)
}

func (qiniu *Qiniu) WriteStream(path string, contents string) error {
	return qiniu.Put(path, contents)
}

func (qiniu *Qiniu) GetVisibility(path string) contracts.FileVisibility {
	if qiniu.private {
		return file.INVISIBLE
	}
	return file.VISIBLE
}

// SetVisibility 七牛不支持修改单个文件的可见性
func (qiniu *Qiniu) SetVisibility(path string, perm fs.FileMode) error {
	return nil
}

func (qiniu *Qiniu) Prepend(path, contents string) error {
	var raw, _ = qiniu.Get(path)
	return qiniu.Put(path, contents+raw)
}

func (qiniu *Qiniu) Append(path, contents string) error {
	var raw, _ = qiniu.Get(path)
	return qiniu.Put(path, raw+contents)
}

func (qiniu *Qiniu) Delete(path string) error {
	return qiniu.bucketManager.Delete(qiniu.bucket, path)
}

func (qiniu *Qiniu) Copy(from, to string) error {
	return qiniu.bucketManager.Copy(qiniu.bucket, from, qiniu.bucket, to, true)
}

func (qiniu *Qiniu) Move(from, to string) error {
	return qiniu.bucketManager.Move(qiniu.bucket, from, qiniu.bucket, to, true)
}

func (qiniu *Qiniu) Size(path string) (int64, error) {
	var stat, err = qiniu.bucketManager.Stat(qiniu.bucket, path)
	if err != nil {
		return 0, err
	}

	return stat.Fsize, nil
}

func (qiniu *Qiniu) LastModified(path string) (time.Time, error) {
	var stat, err = qiniu.bucketManager.Stat(qiniu.bucket, path)
	if err != nil {
		return time.Time{}, err
	}

	return storage.ParsePutTime(stat.PutTime), err
}

func (qiniu *Qiniu) Files(directory string) []contracts.File {
	var (
		limit     = 1000
		delimiter = directory
		marker    = ""
		files     = make([]contracts.File, 0)
	)
	//初始列举marker为空
	for {
		var entries, _, nextMarker, hashNext, err = qiniu.bucketManager.ListFiles(qiniu.bucket, directory, delimiter, marker, limit)
		if err != nil {
			logs.WithError(err).WithField("dir", directory).Debug("Qiniu.Files: ListFiles failed")
			break
		}
		//print entries
		for _, entry := range entries {
			files = append(files, &file.File{
				FileInfo: QiniuFileInfo{
					ListItem: &entry,
					name:     entry.Key,
				},
				DiskName: qiniu.Name(),
			})
		}
		if hashNext {
			marker = nextMarker
		} else {
			//list end
			break
		}
	}
	return files
}

func (qiniu *Qiniu) AllFiles(directory string) []contracts.File {
	var (
		limit     = 1000
		delimiter = ""
		marker    = ""
		files     = make([]contracts.File, 0)
	)
	//初始列举marker为空
	for {
		var entries, _, nextMarker, hashNext, err = qiniu.bucketManager.ListFiles(qiniu.bucket, directory, delimiter, marker, limit)
		if err != nil {
			logs.WithError(err).WithField("dir", directory).Debug("Qiniu.Files: ListFiles failed")
			break
		}
		//print entries
		for _, entry := range entries {
			files = append(files, &file.File{
				FileInfo: QiniuFileInfo{
					ListItem: &entry,
					name:     entry.Key,
				},
				DiskName: qiniu.Name(),
			})
		}
		if hashNext {
			marker = nextMarker
		} else {
			//list end
			break
		}
	}
	return files
}

func (qiniu *Qiniu) Directories(directory string) []string {
	return nil
}

func (qiniu *Qiniu) AllDirectories(directory string) []string {
	return nil
}

func (qiniu *Qiniu) MakeDirectory(path string) error {
	return nil
}

func (qiniu *Qiniu) DeleteDirectory(directory string) error {
	var (
		files = qiniu.Files(directory)
		keys  = make([]string, 0, len(files))
	)
	for _, item := range files {
		keys = append(keys, storage.URIDelete(qiniu.bucket, item.Name()))
	}
	rets, err := qiniu.bucketManager.Batch(keys)
	if err != nil {
		// 遇到错误
		if _, ok := err.(*storage.ErrorInfo); ok {
			for _, ret := range rets {
				// 200 为成功
				if ret.Code != 200 {
					logs.WithError(err).WithField("ret", ret).Debug("Qiniu.DeleteDirectory: delete directory failed")
				}
			}
		} else {
			logs.WithError(err).WithField("rets", rets).Debug("Qiniu.DeleteDirectory: delete directory failed")
		}
		return err
	}
	return nil
}
