# syntax=docker/dockerfile:1

# --- build -----------------------------------------------------------
FROM golang:1.26-alpine AS build
WORKDIR /src

# Dependencies first, so they stay cached until go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG VERSION=docker
RUN CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w -X main.version=${VERSION}" \
      -o /out/natterbot ./cmd/natterbot \
 && mkdir /out/data

# --- runtime ---------------------------------------------------------
# distroless/static: CA certificates and tzdata, no shell, non-root user.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/natterbot /usr/local/bin/natterbot
# Pre-created and owned by the runtime user so a fresh named volume
# mounted here is writable.
COPY --from=build --chown=nonroot:nonroot /out/data /data

WORKDIR /data
ENV SESSION_FILE=/data/session.json
VOLUME /data

ENTRYPOINT ["/usr/local/bin/natterbot"]
