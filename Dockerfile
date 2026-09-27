FROM --platform=$BUILDPLATFORM golang:1.27-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=unknown

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -o /out/bot ./cmd/bot

FROM gcr.io/distroless/static-debian12:nonroot

ARG VERSION=dev
ARG COMMIT=unknown

LABEL org.opencontainers.image.title="discord-bot" \
      org.opencontainers.image.description="Neuland Ingolstadt onboarding ticket Discord bot" \
      org.opencontainers.image.source="https://github.com/neuland-ingolstadt/discord-bot" \
      org.opencontainers.image.licenses="EUPL-1.2" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}"

COPY --from=builder /out/bot /bot

USER nonroot:nonroot
ENTRYPOINT ["/bot"]
