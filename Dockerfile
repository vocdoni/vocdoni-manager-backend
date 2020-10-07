# This first chunk downloads dependencies and builds the binaries, in a way that
# can easily be cached and reused.

FROM golang:1.14.4 AS builder

WORKDIR /src

# We also need the duktape stub for the 'go mod download'. Note that we need two
# COPY lines, since otherwise we do the equivalent of 'cp duktape-stub/* .'.
COPY go.mod go.sum ./
RUN go mod download
RUN apt update && apt install -y ca-certificates 

# Build all the binaries at once, so that the final targets don't require having
# Go installed to build each of them.
COPY . .
RUN go build -o=. -ldflags='-w -s' -mod=readonly ./cmd/dvotemanager ./cmd/managertest ./cmd/dvotenotif

FROM debian:10-slim as managertest

WORKDIR /app
COPY --from=builder /src/managertest ./

ENTRYPOINT ["/app/managertest"]

FROM debian:10-slim as dvotenotif

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
WORKDIR /app
COPY --from=builder /src/dvotenotif ./

ENTRYPOINT ["/app/dvotenotif"]

FROM debian:10-slim

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
WORKDIR /app
COPY --from=builder /src/dvotemanager ./

ENTRYPOINT ["/app/dvotemanager"]

