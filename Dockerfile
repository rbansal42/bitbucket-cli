# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 go build \
    -ldflags "-s -w -X github.com/rbansal42/bitbucket-cli/internal/cmd.Version=${VERSION} -X github.com/rbansal42/bitbucket-cli/internal/cmd.BuildDate=${BUILD_DATE}" \
    -o /bin/bb ./cmd/bb

# Runtime stage
FROM alpine:3.21

LABEL org.opencontainers.image.source="https://github.com/rbansal42/bitbucket-cli"
LABEL org.opencontainers.image.description="Unofficial CLI for Bitbucket Cloud"

RUN apk add --no-cache git ca-certificates && \
    adduser -D -h /home/bb bb

USER bb

COPY --from=builder /bin/bb /usr/local/bin/bb

ENTRYPOINT ["bb"]
