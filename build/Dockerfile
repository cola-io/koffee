ARG BASEIMAGE=alpine:3.21
ARG GOVERSION=1.24.3-bullseye

FROM golang:${GOVERSION} AS builder
COPY . /go/src/cola.io/koffee
WORKDIR /go/src/cola.io/koffee

RUN make all

FROM ${BASEIMAGE}
COPY --from=builder /go/src/cola.io/koffee/bin/koffee /bin/koffee
WORKDIR     /bin
CMD         ["koffee"]
