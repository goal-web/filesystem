package filesystem

import (
	"github.com/goal-web/contracts"
	"github.com/goal-web/filesystem/adapters"
)

type ServiceProvider struct {
}

func (this ServiceProvider) Stop() {

}

func (this ServiceProvider) Start() error {
	return nil
}

func (this ServiceProvider) Register(container contracts.Application) {
	container.Singleton("filesystem", func(config contracts.Config) contracts.FileSystemFactory {
		factory := &Factory{
			config: config,
			disks:  make(map[string]contracts.FileSystem),
			drivers: map[string]contracts.FileSystemProvider{
				"local": adapters.LocalAdapter,
				"qiniu": adapters.QiniuAdapter,
			},
		}

		return factory
	})

	container.Singleton("system.default", func(factory contracts.FileSystemFactory) contracts.FileSystem {
		return factory
	})
}
