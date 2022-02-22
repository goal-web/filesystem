package adapters

import (
	"bufio"
	"fmt"
	"github.com/goal-web/contracts"
	"github.com/goal-web/filesystem/file"
	"github.com/goal-web/supports/utils"
	"io/fs"
	"io/ioutil"
	"os"
	"strings"
	"syscall"
	"time"
)

type local struct {
	name string
	root string
	perm fs.FileMode
}

func LocalAdapter(name string, config contracts.Fields) contracts.FileSystem {
	return NewLocalFileSystem(
		name,
		utils.GetStringField(config, "root"),
		config["perm"].(fs.FileMode),
	)
}

func NewLocalFileSystem(name, root string, perm fs.FileMode) contracts.FileSystem {
	stat, err := os.Stat(root)

	if err != nil {
		err = os.Mkdir(root, perm)
		if err != nil {
			panic(err)
		}
	} else if !stat.IsDir() {
		panic(fmt.Errorf("%s is not a directory", root))
	}

	if !strings.HasSuffix(root, "/") {
		root = root + "/"
	}

	return &local{
		root: root,
		perm: perm,
		name: name,
	}
}

func (this local) filepath(path string) string {
	if strings.HasPrefix(path, "/") {
		runes := []rune(path)
		path = string(runes[1:])
	}
	return this.root + path
}
func (this local) dir(path string) string {
	var arr = strings.Split(strings.ReplaceAll(path, this.root, ""), "/")
	return strings.Join(arr[:len(arr)-1], "/")
}

func (this *local) Name() string {
	return this.name
}

func (this *local) Exists(path string) bool {
	_, err := os.Stat(this.filepath(path))
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func (this *local) Get(path string) (string, error) {
	contents, err := ioutil.ReadFile(this.filepath(path))
	return string(contents), err
}

func (this *local) Read(path string) ([]byte, error) {
	contents, err := ioutil.ReadFile(this.filepath(path))
	return contents, err
}

func (this *local) ReadStream(path string) (*bufio.Reader, error) {
	var f, err = os.Open(this.filepath(path))
	return bufio.NewReader(f), err
}

func (this *local) Put(path, contents string) error {
	path = this.filepath(path)
	var openFile, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, this.perm)
	if err != nil {
		if mkdirErr := this.MakeDirectory(this.dir(path)); mkdirErr != nil {
			return mkdirErr
		}
	}
	_, err = openFile.WriteString(contents)
	return err
}

func (this *local) WriteStream(path string, contents string) error {
	path = this.filepath(path)
	openFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, this.perm)
	if err != nil {
		return err
	}
	defer openFile.Close()
	writer := bufio.NewWriter(openFile)
	_, err = writer.WriteString(contents)
	if err != nil {
		return err
	}
	return writer.Flush()
}

func (this *local) GetVisibility(path string) contracts.FileVisibility {
	if syscall.Access(this.filepath(path), syscall.O_RDWR) != nil {
		return file.INVISIBLE
	}
	return file.VISIBLE
}

func (this *local) SetVisibility(path string, perm fs.FileMode) error {
	return os.Chmod(this.filepath(path), perm)
}

func (this *local) Prepend(path, contents string) error {
	originalData, err := this.Get(path)

	if err != nil {
		return this.WriteStream(path, contents)
	}

	return this.WriteStream(path, contents+originalData)
}

func (this *local) Append(path, contents string) error {
	path = this.filepath(path)
	var openFile, err = os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, os.ModeAppend|this.perm)
	if err != nil {
		return err
	}
	defer openFile.Close()
	_, err = openFile.WriteString(contents)
	return err
}

func (this *local) Delete(path string) error {
	return os.Remove(this.filepath(path))
}

func (this *local) Copy(from, to string) error {
	return utils.CopyFile(this.filepath(from), this.filepath(to), 1000)
}

func (this *local) Move(from, to string) error {
	return os.Rename(this.filepath(from), this.filepath(to))
}

func (this *local) Size(path string) (int64, error) {
	stat, err := os.Stat(this.filepath(path))
	if err != nil {
		return 0, err
	}

	return stat.Size(), nil
}

func (this *local) LastModified(path string) (time.Time, error) {
	stat, err := os.Stat(this.filepath(path))
	if err != nil {
		return time.Time{}, err
	}

	return stat.ModTime(), nil
}

func (this *local) Files(directory string) (results []contracts.File) {
	fileInfos, err := ioutil.ReadDir(this.filepath(directory))
	if err != nil {
		return
	}

	for _, fileInfo := range fileInfos {
		if !fileInfo.IsDir() {
			results = append(results, &File{
				FileInfo: fileInfo,
				DiskName: this.name,
				path:     this.filepath(directory + "/" + fileInfo.Name()),
			})
		}
	}

	return
}

func (this *local) AllFiles(directory string) (results []contracts.File) {
	fileInfos := utils.AllFiles(this.filepath(directory))

	for _, fileInfo := range fileInfos {
		results = append(results, &File{
			FileInfo: fileInfo,
			DiskName: this.name,
			path:     this.filepath(directory + "/" + fileInfo.Name()),
		})
	}

	return
}

func (this *local) Directories(directory string) (results []string) {
	fileInfos, err := ioutil.ReadDir(this.filepath(directory))
	if err != nil {
		return
	}

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			results = append(results, fileInfo.Name())
		}
	}
	return results
}

func (this *local) AllDirectories(directory string) []string {
	return utils.AllDirectories(this.filepath(directory))
}

func (this *local) MakeDirectory(path string) error {
	return os.Mkdir(this.filepath(path), this.perm)
}

func (this *local) DeleteDirectory(directory string) error {
	return os.RemoveAll(this.filepath(directory))
}
