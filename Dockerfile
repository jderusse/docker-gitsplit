FROM golang:alpine AS build

RUN apk add --no-cache \
        git

RUN go get -d github.com/libgit2/git2go
RUN cd $GOPATH/src/github.com/libgit2/git2go \
 && git checkout v27 \
 && git submodule update --init

RUN apk add --no-cache \
        make\
        cmake \
        g++ \
        libressl-dev \
        libssh2-dev \
        libgit2-dev

RUN cd $GOPATH/src/github.com/libgit2/git2go \
 && make install-static

COPY . /go/src/github.com/jderusse/gitsplit/

RUN go get github.com/jderusse/gitsplit
RUN go build -o gitsplit github.com/jderusse/gitsplit

# ==================================================

FROM alpine

RUN apk add --no-cache \
        git \
        openssh-client \
        ca-certificates

COPY --from=build /go/gitsplit /bin/gitsplit

WORKDIR /srv
CMD ["gitsplit"]
