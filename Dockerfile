FROM golang:alpine AS build

RUN apk add --no-cache \
        git

RUN go get -d github.com/libgit2/git2go
RUN cd $GOPATH/src/github.com/libgit2/git2go \
 && git checkout next \
 && git submodule update --init

RUN apk add --no-cache \
        make\
        cmake \
        g++
RUN cd $GOPATH/src/github.com/libgit2/git2go \
 && make install

RUN go get github.com/splitsh/lite
RUN go build -o splitsh-lite github.com/splitsh/lite

# ==================================================

FROM python:3-alpine

RUN apk add --no-cache \
        git \
        openssh-client

RUN pip install pyyaml
ENV PYTHONUNBUFFERED=0

COPY --from=build /go/splitsh-lite /bin/splitsh-lite
COPY gitsplit /bin/gitsplit

WORKDIR /srv
CMD ["gitsplit"]
