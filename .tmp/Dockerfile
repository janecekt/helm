FROM alpine:3.14

RUN mkdir -v /build \
    && apk --no-cache add \
            ca-certificates \
            curl \
            go \
            make \
            bash \
            git \
            strace

ENV GOPATH /build

WORKDIR /build