# Stage 1: Build Go binary
FROM golang:1.26.1 AS go-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/reporter ./cmd/reporter

# Stage 2: Download gws pre-built binary (Rust native, musl static build)
FROM alpine:3.21 AS gws-downloader
ARG GWS_VERSION=0.22.3
RUN apk add --no-cache wget ca-certificates \
  && wget -O /tmp/gws.tar.gz \
     "https://github.com/googleworkspace/cli/releases/download/v${GWS_VERSION}/google-workspace-cli-x86_64-unknown-linux-musl.tar.gz" \
  && tar -xzf /tmp/gws.tar.gz -C /tmp \
  && mv /tmp/google-workspace-cli-x86_64-unknown-linux-musl/gws /usr/local/bin/gws \
  && chmod +x /usr/local/bin/gws

# Stage 3: Distroless runtime (no shell, no package manager)
FROM gcr.io/distroless/base-debian12@sha256:937c7eaaf6f3f2d38a1f8c4aeff326f0c56e4593ea152e9e8f74d976dde52f56
COPY --from=go-builder /app/reporter /app/reporter
COPY --from=gws-downloader /usr/local/bin/gws /usr/local/bin/gws
ENTRYPOINT ["/app/reporter"]
