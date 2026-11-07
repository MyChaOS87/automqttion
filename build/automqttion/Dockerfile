FROM golang:1.27-alpine AS builder

RUN apk update && apk add --no-cache git ca-certificates tzdata musl-utils && update-ca-certificates

ENV USER=appuser
ENV UID=10001

RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY pkg ./pkg
COPY internal ./internal
COPY config ./config
COPY cmd ./cmd

RUN \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -v -ldflags "-w -s -X main.Version=$APP_VERSION -extldflags '-static'" -o /dist/automqttion ./cmd/automqttion
RUN ldd /dist/automqttion | tr -s '[:blank:]' '\n' | grep '^/' | \
    xargs -I % sh -c 'mkdir -p $(dirname /dist%); cp % /dist%;'


############################
# RUN IMAGE
############################
FROM scratch

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /dist /
COPY --from=builder /build/config/config.yml /config/

USER appuser:appuser

ENTRYPOINT ["/automqttion"]
