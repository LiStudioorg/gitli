# SPDX-License-Identifier: AGPL-3.0
# 多阶段构建；最终镜像基于 alpine，gitli 本身为纯静态二进制（CGO_ENABLED=0）。
# 构建：docker build -f deploy/docker/Dockerfile -t gitli .
#
# 多架构说明：前端与 Go 编译阶段固定跑在 BUILDPLATFORM（构建机原生架构），
# 仅用 TARGETOS/TARGETARCH 交叉编译出目标二进制。这样 arm64 镜像无需 QEMU
# 模拟 node 与 go 的编译过程，是多架构构建提速的关键。

FROM --platform=$BUILDPLATFORM node:22-alpine AS frontend
WORKDIR /src/web
COPY web/package.json web/package-lock.json* ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS backend
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /src/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags "-s -w" -o /out/gitli .

FROM alpine:3.19
# 唯一运行时外部依赖是系统 git（用于 Git 操作）。
RUN apk add --no-cache git openssh ca-certificates \
    && adduser -D -H -s /sbin/nologin gitli
WORKDIR /var/lib/gitli
COPY --from=backend /out/gitli /usr/local/bin/gitli
EXPOSE 8080 2222
VOLUME ["/var/lib/gitli"]
USER gitli
ENTRYPOINT ["/usr/local/bin/gitli"]
CMD ["serve", "--config", "/etc/gitli/config.toml"]
