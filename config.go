package filesystem

import "github.com/goal-web/contracts"

type Config struct {
	Default string

	Disks map[string]contracts.Fields
}
