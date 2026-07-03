# =============================================================================
# 天眼 (empyrean_lens) Dockerfile
# 三阶段构建：前端 → Go 编译 → 最终镜像
# =============================================================================

# 第一阶段：构建前端（web/dist 由 embed.FS 打进 Go 二进制）
FROM node:22-alpine AS web-builder

WORKDIR /app/web
COPY web/package*.json ./
RUN npm install
COPY web/ ./
RUN npm run build

# 第二阶段：编译 Go 二进制
FROM golang:1.23-alpine AS builder

ENV GOPROXY="https://goproxy.cn,direct"
ENV GOPRIVATE="codeup.aliyun.com"
ENV GONOSUMDB="*"
ENV GOFLAGS="-mod=mod"

RUN sed -i 's#dl-cdn.alpinelinux.org#mirrors.aliyun.com#g' /etc/apk/repositories \
    && apk update \
    && apk add --no-cache git ca-certificates

WORKDIR /app

# 先复制依赖文件，利用 Docker 层缓存
COPY go.mod go.sum ./

# codeup.aliyun.com 私有仓库认证（通过 build-arg 传入）
ARG CODEUP_USER
ARG CODEUP_PASSWORD
RUN if [ -n "$CODEUP_USER" ] && [ -n "$CODEUP_PASSWORD" ]; then \
        echo "machine codeup.aliyun.com login ${CODEUP_USER} password ${CODEUP_PASSWORD}" >> ~/.netrc; \
    fi

RUN go mod download

# 复制源代码
COPY . .

# 用前端构建产物覆盖（embed.FS 需要此目录存在且完整）
COPY --from=web-builder /app/web/dist ./web/dist

# 静态链接，embed.FS 将 web/dist 打入二进制
RUN go build -ldflags "-s -w -extldflags '-static'" -o app .

# 第三阶段：最终运行镜像
FROM debian:bookworm-slim

ENV TZ=Asia/Shanghai

RUN sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list.d/debian.sources \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        tzdata \
        vim \
        wget \
        curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /opt/output

COPY --from=builder /app/app ./

RUN mkdir -p logs

EXPOSE 8080

# MODE_ENV 通过运行时环境变量设置（dev / pre / prod）
CMD ["./app"]
