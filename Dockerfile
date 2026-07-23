# Build stage
FROM golang:1.25.0-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git protobuf protobuf-dev

# 容器内默认 GOPROXY 为 proxy.golang.org，国内网络通常不可达，
# 显式设置为本机一致的国内镜像，避免 go mod download 失败。
ENV GOPROXY=https://goproxy.cn,https://goproxy.io,direct
ENV GOSUMDB=off

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Generate proto code
RUN protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/comment.proto

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o comment-service ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary
COPY --from=builder /app/comment-service .
COPY --from=builder /app/config.yaml .

EXPOSE 8083 9003 9093

CMD ["./comment-service"]
