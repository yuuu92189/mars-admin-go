# syntax=docker/dockerfile:1.7
ARG RUNTIME_IMAGE=gcr.io/distroless/static-debian13:nonroot

FROM golang:1.27.3-bookworm AS builder

WORKDIR /src
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOPROXY=https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN mkdir -p /out/uploads \
 && go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker \
 && go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate \
 && go build -trimpath -ldflags="-s -w" -o /out/legacy-import ./cmd/legacy-import

FROM ${RUNTIME_IMAGE}

WORKDIR /app
COPY --from=builder /out/api /usr/local/bin/api
COPY --from=builder /out/worker /usr/local/bin/worker
COPY --from=builder /out/migrate /usr/local/bin/migrate
COPY --from=builder /out/legacy-import /usr/local/bin/legacy-import
COPY --from=builder /src/api/openapi/v1/openapi.yaml /app/openapi.yaml
COPY --from=builder --chown=65532:65532 /out/uploads /app/uploads

USER 65532:65532
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 CMD ["/usr/local/bin/api", "-healthcheck"]
ENTRYPOINT ["/usr/local/bin/api"]
