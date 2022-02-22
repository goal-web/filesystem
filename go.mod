module github.com/goal-web/filesystem

go 1.17

require (
	github.com/goal-web/contracts v0.1.49
	github.com/goal-web/supports v0.1.16
	github.com/qiniu/go-sdk/v7 v7.11.1
	github.com/stretchr/testify v1.7.0
)

require (
	github.com/apex/log v1.9.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pkg/errors v0.8.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace github.com/goal-web/contracts => ../contracts
