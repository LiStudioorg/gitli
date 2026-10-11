# SPDX-License-Identifier: AGPL-3.0
FROM node:20-alpine AS frontend
WORKDIR /src/web
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web/ ./
RUN npm run build

FROM golang:1.23-alpine AS backend
WORKDIR /src
RUN apk add --no-cache git openssh
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /src/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -o /out/gitli .

FROM alpine:3.20
RUN apk add --no-cache git openssh ca-certificates && adduser -D -H -s /sbin/nologin gitli
WORKDIR /var/lib/gitli
COPY --from=backend /out/gitli /usr/local/bin/gitli
EXPOSE 8080 2222
VOLUME ["/var/lib/gitli"]
ENTRYPOINT ["/usr/local/bin/gitli"]
CMD ["serve", "--config", "/etc/gitli/config.toml"]
