FROM golang:alpine AS builder

RUN apk update && apk add --no-cache git ca-certificates tzdata musl-utils && update-ca-certificates
WORKDIR /build
COPY vendor ./vendor
COPY go.mod go.sum ./
COPY pkg ./pkg
COPY internal ./internal
COPY config ./config
COPY cmd ./cmd

RUN \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -v -ldflags "-X main.Version=$APP_VERSION -extldflags '-static'" -o /dist/automqttion ./cmd/automqttion
RUN ldd /dist/automqttion | tr -s '[:blank:]' '\n' | grep '^/' | \
    xargs -I % sh -c 'mkdir -p $(dirname /dist%); cp % /dist%;'


############################
# RUN IMAGE
############################
FROM scratch

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /dist /
COPY --from=builder /build/config/config.yml /config/

ENTRYPOINT ["/automqttion"]