package adapters

import (
	"bufio"
	"fmt"
	"github.com/goal-web/contracts"
	"github.com/goal-web/filesystem/file"
	"github.com/goal-web/supports/utils"
	"io/fs"
	"os"
	"strings"
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

func (localAdapter local) filepath(path string) string {
	if strings.HasPrefix(path, "/") {
		runes := []rune(path)
		path = string(runes[1:])
	}
	return localAdapter.root + path
}
func (localAdapter local) dir(path string) string {
	var arr = strings.Split(strings.ReplaceAll(path, localAdapter.root, ""), "/")
	return strings.Join(arr[:len(arr)-1], "/")
}

func (localAdapter *local) Name() string {
	return localAdapter.name
}

func (localAdapter *local) Exists(path string) bool {
	_, err := os.Lstat(localAdapter.filepath(path))
	return !os.IsNotExist(err)
}

func (localAdapter *local) Get(path string) (string, error) {
	contents, err := os.ReadFile(localAdapter.filepath(path))
	return string(contents), err
}

func (localAdapter *local) Read(path string) ([]byte, error) {
	contents, err := os.ReadFile(localAdapter.filepath(path))
	return contents, err
}

func (localAdapter *local) ReadStream(path string) (*bufio.Reader, error) {
	var f, err = os.Open(localAdapter.filepath(path))
	return bufio.NewReader(f), err
}

func (localAdapter *local) Put(path, contents string) error {
	path = localAdapter.filepath(path)
	var openFile, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, localAdapter.perm)
	if err != nil {
		if mkdirErr := localAdapter.MakeDirectory(localAdapter.dir(path)); mkdirErr != nil {
			return mkdirErr
		}
	}
	_, err = openFile.WriteString(contents)
	return err
}

func (localAdapter *local) WriteStream(path string, contents string) error {
	path = localAdapter.filepath(path)
	openFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, localAdapter.perm)
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

func (localAdapter *local) GetVisibility(path string) contracts.FileVisibility {
	openFile, err := os.OpenFile(localAdapter.filepath(path), os.O_RDWR, localAdapter.perm)
	if openFile != nil && err == nil {
		_ = openFile.Close()
		return file.VISIBLE
	}
	return file.INVISIBLE
}

func (localAdapter *local) SetVisibility(path string, perm fs.FileMode) error {
	return os.Chmod(localAdapter.filepath(path), perm)
}

func (localAdapter *local) Prepend(path, contents string) error {
	originalData, err := localAdapter.Get(path)

	if err != nil {
		return localAdapter.WriteStream(path, contents)
	}

	return localAdapter.WriteStream(path, contents+originalData)
}

func (localAdapter *local) Append(path, contents string) error {
	path = localAdapter.filepath(path)
	var openFile, err = os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, os.ModeAppend|localAdapter.perm)
	if err != nil {
		return err
	}
	defer openFile.Close()
	_, err = openFile.WriteString(contents)
	return err
}

func (localAdapter *local) Delete(path string) error {
	return os.Remove(localAdapter.filepath(path))
}

func (localAdapter *local) Copy(from, to string) error {
	return utils.CopyFile(localAdapter.filepath(from), localAdapter.filepath(to), 1000)
}

func (localAdapter *local) Move(from, to string) error {
	return os.Rename(localAdapter.filepath(from), localAdapter.filepath(to))
}

func (localAdapter *local) Size(path string) (int64, error) {
	stat, err := os.Stat(localAdapter.filepath(path))
	if err != nil {
		return 0, err
	}

	return stat.Size(), nil
}

func (localAdapter *local) LastModified(path string) (time.Time, error) {
	stat, err := os.Stat(localAdapter.filepath(path))
	if err != nil {
		return time.Time{}, err
	}

	return stat.ModTime(), nil
}

func (localAdapter *local) Files(directory string) (results []contracts.File) {
	fileInfos, err := os.ReadDir(localAdapter.filepath(directory))
	if err != nil {
		return
	}

	for _, fileInfo := range fileInfos {
		if !fileInfo.IsDir() {
			info, _ := fileInfo.Info()
			results = append(results, &File{
				FileInfo: info,
				DiskName: localAdapter.name,
				path:     localAdapter.filepath(directory + "/" + fileInfo.Name()),
			})
		}
	}

	return
}

func (localAdapter *local) AllFiles(directory string) (results []contracts.File) {
	fileInfos := utils.AllFiles(localAdapter.filepath(directory))

	for _, fileInfo := range fileInfos {
		results = append(results, &File{
			FileInfo: fileInfo,
			DiskName: localAdapter.name,
			path:     localAdapter.filepath(directory + "/" + fileInfo.Name()),
		})
	}

	return
}

func (localAdapter *local) Directories(directory string) (results []string) {
	fileInfos, err := os.ReadDir(localAdapter.filepath(directory))
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

func (localAdapter *local) AllDirectories(directory string) []string {
	return utils.AllDirectories(localAdapter.filepath(directory))
}

func (localAdapter *local) MakeDirectory(path string) error {
	return os.Mkdir(localAdapter.filepath(path), localAdapter.perm)
}

func (localAdapter *local) DeleteDirectory(directory string) error {
	return os.RemoveAll(localAdapter.filepath(directory))
}
