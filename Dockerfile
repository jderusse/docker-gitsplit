FROM golang:alpine AS libgit2

RUN apk add --no-cache \
        git \
        g++ \
        make \
        cmake \
        openssl-dev \
        libssh2-dev

RUN go get -d github.com/libgit2/git2go \
 && cd $GOPATH/src/github.com/libgit2/git2go \
 && git submodule update --init \
 && make install-static

# ==================================================

FROM golang:alpine AS build

RUN apk add --no-cache \
        git \
        g++ \
        libssh2-dev

COPY --from=libgit2 /go /go

WORKDIR /go/src/github.com/jderusse/gitsplit/
COPY . /go/src/github.com/jderusse/gitsplit/

RUN go get --tags "static" ./
RUN go build --tags "static" -o /go/bin/gitsplit ./

# ==================================================

FROM alpine:3.8

RUN apk add --no-cache \
        git \
        openssl \
        openssh-client \
        ca-certificates

COPY --from=build /go/bin/gitsplit /bin/gitsplit

WORKDIR /srv
CMD ["gitsplit"]
