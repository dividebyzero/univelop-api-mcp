FROM golang:1.27-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(git describe --tags --always 2>/dev/null || echo dev)" \
    -o /univelop-mcp ./cmd/univelop-mcp/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /univelop-mcp /usr/local/bin/univelop-mcp
EXPOSE 8443
ENTRYPOINT ["univelop-mcp"]
CMD ["--config", "/etc/univelop-mcp/config.yaml"]