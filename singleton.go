package filesystem

import (
	"sync"

	"github.com/goal-web/application"
	"github.com/goal-web/contracts"
)

var singleton contracts.FileSystemFactory
var once sync.Once

func Default() contracts.FileSystemFactory {
	once.Do(func() {
		singleton = application.Get("filesystem").(contracts.FileSystemFactory)
	})

	return singleton
}

func Disk(name string) contracts.FileSystem {

	return Default().Disk(name)
}
