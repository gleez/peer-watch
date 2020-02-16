############################
# STEP 1 build executable binary
############################
FROM golang:1.13 as builder

ADD . /go/src/github.com/gleez/peer-watch

RUN cd /go/src/github.com/gleez/peer-watch \
 && COMMIT_SHA=$(git rev-parse --short HEAD) \
 && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w  \
    -X main.Version=0.3 \
    -X main.Revision=${COMMIT_SHA}" \
    -a -tags netgo -installsuffix netgo -o peer-watch main.go
    
############################
# STEP 2 build a certs image
############################

# Alpine certs
FROM alpine:3.11 as alpine

RUN apk update && apk add --no-cache ca-certificates tzdata && update-ca-certificates

# Create appuser
ENV USER=appuser
ENV UID=10001

# See https://stackoverflow.com/a/55757473/12429735
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"


############################
# STEP 3 build a release image
############################
FROM scratch
MAINTAINER Sandeep Sangamreddi <sandeepone@gmail.com>

# Import from builder.
COPY --from=alpine /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=alpine /etc/passwd /etc/passwd
COPY --from=alpine /etc/group /etc/group

# Add the binary
COPY --from=builder /go/src/github.com/gleez/peer-watch/peer-watch /usr/bin/

EXPOSE 4040

# Use an unprivileged user.
USER appuser:appuser

ENTRYPOINT ["peer-watch"]