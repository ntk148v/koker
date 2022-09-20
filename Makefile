build:
	go build -ldflags "-s -w -X main.version=`git tag --sort=-version:refname | head -n 1`" -o koker cmd/koker/main.go
run:
	sudo go run cmd/koker/main.go
