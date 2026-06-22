# SPDX-License-Identifier: MIT

FROM golang:1.22-alpine AS build

WORKDIR /src
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /out/idbridge ./cmd/idbridge
RUN CGO_ENABLED=0 go build -o /out/goose github.com/pressly/goose/v3/cmd/goose

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=build /out/idbridge /usr/local/bin/idbridge
COPY --from=build /out/goose /usr/local/bin/goose
COPY migrations /app/migrations

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/idbridge"]
