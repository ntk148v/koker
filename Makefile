GOOS ?= linux
GOARCH ?= amd64

build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "-s -w -X main.version=`git tag --sort=-version:refname | head -n 1`" -o /tmp/koker cmd/koker/main.go

test:
	go test -v ./pkg/utils ./pkg/filesystem ./pkg/cgroups ./pkg/images
	GOOS=linux go vet ./...

run:
	sudo go run cmd/koker/main.go
