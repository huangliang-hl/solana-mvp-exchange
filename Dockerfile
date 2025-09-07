# 多阶段构建Dockerfile - 统一构建所有服务
# 使用方式：
# docker build --target api -t solana-exchange/api .
# docker build --target price -t solana-exchange/price .
# docker build --target worker -t solana-exchange/worker .

# 构建阶段
FROM golang:1.23-alpine AS builder

# 安装必要的构建工具
RUN apk add --no-cache git ca-certificates tzdata

# 设置工作目录
WORKDIR /app

# 复制go mod文件并下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建所有服务
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/price ./cmd/price && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/worker ./cmd/worker

# API服务运行阶段
FROM alpine:latest AS api

# 安装必要的运行时依赖
RUN apk --no-cache add ca-certificates tzdata curl

WORKDIR /root/

# 从构建阶段复制API二进制文件
COPY --from=builder /app/bin/api .

# 创建日志目录
RUN mkdir -p /app/logs

# 暴露端口
EXPOSE 8080

# 设置健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

# 运行API服务
CMD ["./api"]

# 价格服务运行阶段
FROM alpine:latest AS price

# 安装必要的运行时依赖
RUN apk --no-cache add ca-certificates tzdata curl

WORKDIR /root/

# 从构建阶段复制Price二进制文件
COPY --from=builder /app/bin/price .

# 创建日志目录
RUN mkdir -p /app/logs

# 暴露端口
EXPOSE 8081

# 设置健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8081/health || exit 1

# 运行价格服务
CMD ["./price"]

# 执行器服务运行阶段
FROM alpine:latest AS worker

# 安装必要的运行时依赖
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# 从构建阶段复制Worker二进制文件
COPY --from=builder /app/bin/worker .

# 创建日志目录
RUN mkdir -p /app/logs

# 运行执行器服务
CMD ["./worker"]

# 测试运行阶段
FROM golang:1.23-alpine AS test

# 安装测试所需的工具
RUN apk add --no-cache git ca-certificates curl postgresql-client redis

WORKDIR /app

# 复制源代码
COPY . .

# 下载依赖
RUN go mod download

# 运行测试
CMD ["go", "test", "-v", "./...", "-timeout=300s"]
