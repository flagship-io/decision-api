FROM golang:1.17-alpine as builder
RUN apk add --update make
WORKDIR /go/src/github/flagship-io/decision-api
COPY . .
RUN make build

FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/github/flagship-io/decision-api/bin/server ./
CMD ["./server"]