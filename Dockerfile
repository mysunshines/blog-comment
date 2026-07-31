# =============================================================================
# comment-service 多阶段构建
#
# 第一阶段 (builder)：编译 Go 源码为二进制产物
# 第二阶段 (final)：  最小化 alpine 运行镜像
# =============================================================================

# 启用 BuildKit 前端语法（支持 --mount=type=cache 等高级特性）
# syntax=docker/dockerfile:1

# =============================================================================
# 第一阶段：编译
# =============================================================================
FROM golang:1.25.0-alpine AS builder

WORKDIR /app

# git 用于 go mod download 拉取私有仓库；ca-certificates 用于 HTTPS 连接
RUN apk add --no-cache git ca-certificates

# Go 模块代理（优先国内镜像加速下载）
ENV GOPROXY=https://goproxy.cn,https://goproxy.io,direct
# 跳过校验和数据库验证（生产环境建议开启）
ENV GOSUMDB=off

# 先复制依赖声明文件，利用 Docker 层缓存：依赖不变时无需重新下载
COPY go.mod go.sum ./
# --mount=type=cache 将模块缓存目录挂载到宿主机，增量构建时复用已下载的依赖
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# 复制完整源码（依赖未变时，只有源码层需要重新构建）
COPY . .

# 构建参数：通过 docker build --build-arg GIT_VERSION=v1.0 传入版本号
ARG GIT_VERSION=dev

ARG APP_NAME=comment-service
# -ldflags 在编译时将版本号注入二进制文件
# --mount=type=cache 分别缓存模块和编译中间产物，大幅加速增量构建
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-X main.Version=${GIT_VERSION}" \
    -o /app/${APP_NAME} ./cmd/server

# =============================================================================
# 第二阶段：运行镜像
# =============================================================================
FROM alpine:latest

ENV APP_NAME=comment-service

# ca-certificates：信任 HTTPS 证书；tzdata：支持时区设置
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# 从构建阶段复制二进制文件和配置目录
COPY --from=builder /app/${APP_NAME} .
COPY --from=builder /app/config/ ./config/

# 8083: HTTP/管理接口, 9003: gRPC 业务通信, 9093: Prometheus Metrics
EXPOSE 8083 9003 9093

# exec 使进程 PID=1，确保能收到 SIGTERM 信号实现优雅停机
CMD exec ./${APP_NAME}
