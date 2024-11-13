FROM golang:1.23.3-alpine as builder
RUN apk add --update make
WORKDIR /go/src/github/flagship-io/decision-api

# Download dependencies before building
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

ARG VERSION
ENV VERSION $VERSION
RUN make build

FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/github/flagship-io/decision-api/bin/server ./
CMD ["./server"]