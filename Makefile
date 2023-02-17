test: SHELL:=/bin/bash
test:
	mkdir -p coverage
	go test -race -coverpkg=./... ./... -coverprofile cover.out.tmp -covermode=atomic
	cat coverage/cover.out.tmp | grep -v "mock_\|examples" > coverage/cover.out
	go tool cover -html=coverage/cover.out -o coverage/cover.html
	go tool cover -func=coverage/cover.out

run:
	go run ./cmd/server/. ${ARGS}

swagger:
	${GOPATH}/bin/swag init -g pkg/server/server.go

build:
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-X 'github.com/flagship-io/decision-api/pkg/models.Version=${VERSION}'" -o bin/server cmd/server/*.go