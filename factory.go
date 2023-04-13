package filesystem

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/goal-web/contracts"
	"github.com/goal-web/filesystem/adapters"
	"github.com/goal-web/supports/exceptions"
	"github.com/goal-web/supports/logs"
	"github.com/goal-web/supports/utils"
	"io/fs"
	"time"
)

var (
	UndefinedDefineErr = errors.New("unsupported file system")
)

func New(config Config) contracts.FileSystemFactory {
	return &Factory{
		config: config,
		disks:  make(map[string]contracts.FileSystem),
		drivers: map[string]contracts.FileSystemProvider{
			"local": adapters.LocalAdapter,
			"qiniu": adapters.QiniuAdapter,
		},
	}

}

type Factory struct {
	config  Config
	disks   map[string]contracts.FileSystem
	drivers map[string]contracts.FileSystemProvider
}

func (this *Factory) Disk(name string) contracts.FileSystem {
	if disk, existsStore := this.disks[name]; existsStore {
		return disk
	}

	this.disks[name] = this.get(name)

	return this.disks[name]
}

func (this *Factory) Extend(driver string, provider contracts.FileSystemProvider) {
	this.drivers[driver] = provider
}

func (this *Factory) get(name string) contracts.FileSystem {
	var (
		config = this.config.Disks[name]
		driver = utils.GetStringField(config, "driver", this.config.Default)
	)
	var driveProvider, existsProvider = this.drivers[driver]
	if !existsProvider {
		logs.WithError(UndefinedDefineErr).Error(fmt.Sprintf("filesystem.Factory: unsupported file system %s", driver))
		panic(Exception{exceptions.WithError(UndefinedDefineErr)})
	}
	return driveProvider(name, config)
}

func (this *Factory) Name() string {
	return this.Disk(this.config.Default).Name()
}

func (this *Factory) Exists(path string) bool {
	return this.Disk(this.config.Default).Exists(path)
}

func (this *Factory) Get(path string) (string, error) {
	return this.Disk(this.config.Default).Get(path)
}

func (this *Factory) Read(path string) ([]byte, error) {
	return this.Disk(this.config.Default).Read(path)
}

func (this *Factory) ReadStream(path string) (*bufio.Reader, error) {
	return this.Disk(this.config.Default).ReadStream(path)
}

func (this *Factory) Put(path, contents string) error {
	return this.Disk(this.config.Default).Put(path, contents)
}

func (this *Factory) WriteStream(path string, contents string) error {
	return this.Disk(this.config.Default).WriteStream(path, contents)
}

func (this *Factory) GetVisibility(path string) contracts.FileVisibility {
	return this.Disk(this.config.Default).GetVisibility(path)
}

func (this *Factory) SetVisibility(path string, perm fs.FileMode) error {
	return this.Disk(this.config.Default).SetVisibility(path, perm)
}

func (this *Factory) Prepend(path, contents string) error {
	return this.Disk(this.config.Default).Prepend(path, contents)
}

func (this *Factory) Append(path, contents string) error {
	return this.Disk(this.config.Default).Append(path, contents)
}

func (this *Factory) Delete(path string) error {
	return this.Disk(this.config.Default).Delete(path)
}

func (this *Factory) Copy(from, to string) error {
	return this.Disk(this.config.Default).Copy(from, to)
}

func (this *Factory) Move(from, to string) error {
	return this.Disk(this.config.Default).Move(from, to)
}

func (this *Factory) Size(path string) (int64, error) {
	return this.Disk(this.config.Default).Size(path)
}

func (this *Factory) LastModified(path string) (time.Time, error) {
	return this.Disk(this.config.Default).LastModified(path)
}

func (this *Factory) Files(directory string) []contracts.File {
	return this.Disk(this.config.Default).Files(directory)
}

func (this *Factory) AllFiles(directory string) []contracts.File {
	return this.Disk(this.config.Default).AllFiles(directory)
}

func (this *Factory) Directories(directory string) []string {
	return this.Disk(this.config.Default).Directories(directory)
}

func (this *Factory) AllDirectories(directory string) []string {
	return this.Disk(this.config.Default).AllDirectories(directory)
}

func (this *Factory) MakeDirectory(path string) error {
	return this.Disk(this.config.Default).MakeDirectory(path)
}

func (this *Factory) DeleteDirectory(directory string) error {
	return this.Disk(this.config.Default).DeleteDirectory(directory)
}
