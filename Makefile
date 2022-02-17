test: SHELL:=/bin/bash
test:
	mkdir -p coverage
	TEST=1 go test -v -race ./... -coverprofile coverage/cover.out.tmp -coverpkg=./... -run .*
	cat coverage/cover.out.tmp | grep -v "mock_" > coverage/cover.out
	go tool cover -html=coverage/cover.out -o coverage/cover.html
	go tool cover -func=coverage/cover.out

run:
	go run *.go ${ARGS}

build:
	env go build -ldflags="-s -w" -o bin/server cmd/server/*.go