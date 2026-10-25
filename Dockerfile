# syntax=docker/dockerfile:1
# One Dockerfile for every service:
#   docker build --platform linux/arm64 --build-arg SERVICE=hello .
# Images are linux/arm64 (ADR 0004): always pass --platform, because BuildKit sets
# TARGETOS/TARGETARCH from it. The build stage runs on the host and cross-compiles.

FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
ARG SERVICE
ARG TARGETOS
ARG TARGETARCH
RUN test -n "$SERVICE" || (echo "build arg SERVICE is required" && exit 1)
RUN test "$TARGETARCH" = "arm64" || (echo "images are linux/arm64 (ADR 0004): build with --platform linux/arm64" && exit 1)

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg
RUN test -d "./cmd/$SERVICE" || (echo "unknown service: $SERVICE" && exit 1)
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/app "./cmd/$SERVICE"

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /app
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app"]
