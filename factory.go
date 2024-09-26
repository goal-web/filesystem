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
	"sync"
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
	mutex   sync.Mutex
	disks   map[string]contracts.FileSystem
	drivers map[string]contracts.FileSystemProvider
}

func (factory *Factory) Disk(name string) contracts.FileSystem {
	if disk, existsStore := factory.disks[name]; existsStore {
		return disk
	}

	factory.mutex.Lock()
	factory.disks[name] = factory.get(name)
	factory.mutex.Unlock()

	return factory.disks[name]
}

func (factory *Factory) Extend(driver string, provider contracts.FileSystemProvider) {
	factory.mutex.Lock()
	defer factory.mutex.Unlock()
	factory.drivers[driver] = provider
}

func (factory *Factory) get(name string) contracts.FileSystem {
	var (
		config = factory.config.Disks[name]
		driver = utils.GetStringField(config, "driver", factory.config.Default)
	)
	var driveProvider, existsProvider = factory.drivers[driver]
	if !existsProvider {
		logs.WithError(UndefinedDefineErr).Error(fmt.Sprintf("filesystem.Factory: unsupported file system %s", driver))
		panic(Exception{exceptions.WithError(UndefinedDefineErr)})
	}
	return driveProvider(name, config)
}

func (factory *Factory) Name() string {
	return factory.Disk(factory.config.Default).Name()
}

func (factory *Factory) Exists(path string) bool {
	return factory.Disk(factory.config.Default).Exists(path)
}

func (factory *Factory) Get(path string) (string, error) {
	return factory.Disk(factory.config.Default).Get(path)
}

func (factory *Factory) Read(path string) ([]byte, error) {
	return factory.Disk(factory.config.Default).Read(path)
}

func (factory *Factory) ReadStream(path string) (*bufio.Reader, error) {
	return factory.Disk(factory.config.Default).ReadStream(path)
}

func (factory *Factory) Put(path, contents string) error {
	return factory.Disk(factory.config.Default).Put(path, contents)
}

func (factory *Factory) WriteStream(path string, contents string) error {
	return factory.Disk(factory.config.Default).WriteStream(path, contents)
}

func (factory *Factory) GetVisibility(path string) contracts.FileVisibility {
	return factory.Disk(factory.config.Default).GetVisibility(path)
}

func (factory *Factory) SetVisibility(path string, perm fs.FileMode) error {
	return factory.Disk(factory.config.Default).SetVisibility(path, perm)
}

func (factory *Factory) Prepend(path, contents string) error {
	return factory.Disk(factory.config.Default).Prepend(path, contents)
}

func (factory *Factory) Append(path, contents string) error {
	return factory.Disk(factory.config.Default).Append(path, contents)
}

func (factory *Factory) Delete(path string) error {
	return factory.Disk(factory.config.Default).Delete(path)
}

func (factory *Factory) Copy(from, to string) error {
	return factory.Disk(factory.config.Default).Copy(from, to)
}

func (factory *Factory) Move(from, to string) error {
	return factory.Disk(factory.config.Default).Move(from, to)
}

func (factory *Factory) Size(path string) (int64, error) {
	return factory.Disk(factory.config.Default).Size(path)
}

func (factory *Factory) LastModified(path string) (time.Time, error) {
	return factory.Disk(factory.config.Default).LastModified(path)
}

func (factory *Factory) Files(directory string) []contracts.File {
	return factory.Disk(factory.config.Default).Files(directory)
}

func (factory *Factory) AllFiles(directory string) []contracts.File {
	return factory.Disk(factory.config.Default).AllFiles(directory)
}

func (factory *Factory) Directories(directory string) []string {
	return factory.Disk(factory.config.Default).Directories(directory)
}

func (factory *Factory) AllDirectories(directory string) []string {
	return factory.Disk(factory.config.Default).AllDirectories(directory)
}

func (factory *Factory) MakeDirectory(path string) error {
	return factory.Disk(factory.config.Default).MakeDirectory(path)
}

func (factory *Factory) DeleteDirectory(directory string) error {
	return factory.Disk(factory.config.Default).DeleteDirectory(directory)
}
