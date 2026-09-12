# syntax=docker/dockerfile:1
# 说明：go.mod 声明 go 1.25.11，因此基础镜像用 golang:1.25-alpine（用 1.22 会因
# go.mod 最低版本要求构建失败）。构建全程 CGO_ENABLED=0，产出静态单二进制。

FROM golang:1.25-alpine AS build
WORKDIR /src

ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/gitli ./cmd/gitli

FROM alpine:3.19

# git：运行时 shell 调用 git upload-pack / receive-pack 等命令所必需
RUN apk add --no-cache git ca-certificates tzdata

COPY --from=build /out/gitli /usr/local/bin/gitli

EXPOSE 3000 2222 9418
VOLUME /data

ENV GITLI_DATA_DIR=/data
ENTRYPOINT ["gitli"]
CMD ["serve"]
