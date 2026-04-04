# Stage 1: Build Go binary
FROM golang:1.26.1 AS go-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -buildvcs=false -o /app/reporter ./cmd/reporter

# Stage 2: Download gws pre-built binary (Rust native, musl static build)
FROM alpine:3.21 AS downloader
ARG GWS_VERSION=0.22.3
RUN apk add --no-cache curl tar
RUN curl -L "https://github.com/googleworkspace/cli/releases/download/v${GWS_VERSION}/google-workspace-cli-x86_64-unknown-linux-musl.tar.gz" \
    | tar -xz -C /tmp \
    && mv /tmp/google-workspace-cli-*/gws /gws

# Stage 3: Distroless static runtime (CA certs only; smaller than base — no glibc for fully static binaries)
FROM gcr.io/distroless/static-debian12@sha256:20bc6c0bc4d625a22a8fde3e55f6515709b32055ef8fb9cfbddaa06d1760f838
COPY --from=go-builder /app/reporter /app/reporter
COPY --from=downloader /gws /usr/local/bin/gws

# Optional: Run as non-root for security
USER 65532:65532

ENTRYPOINT ["/app/reporter"]
