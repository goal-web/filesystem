package filesystem

import (
	"github.com/goal-web/contracts"
	"github.com/goal-web/filesystem/adapters"
)

type ServiceProvider struct {
}

func NewService() contracts.ServiceProvider {
	return &ServiceProvider{}
}

func (provider ServiceProvider) Stop() {

}

func (provider ServiceProvider) Start() error {
	return nil
}

func (provider ServiceProvider) Register(container contracts.Application) {
	container.Singleton("filesystem", func(config contracts.Config) contracts.FileSystemFactory {
		return New(config.Get("filesystem").(Config))
	})

	container.Singleton("fs.default", func(factory contracts.FileSystemFactory) contracts.FileSystem {
		return factory
	})

	container.Singleton("system.qiniu", func(factory contracts.FileSystemFactory) *adapters.Qiniu {
		var adapter, _ = factory.Disk("qiniu").(*adapters.Qiniu)
		return adapter
	})
}
