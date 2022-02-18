FROM golang:1.17-alpine as builder
WORKDIR /go/src/github/flagship-io/decision-api
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server cmd/server/server.go

FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/github/flagship-io/decision-api/server ./
CMD ["./server"]