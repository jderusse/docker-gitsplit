FROM golang

RUN apt-get update \
 && apt-get install -y cmake pkg-config \

 && go get -d github.com/libgit2/git2go \
 && cd $GOPATH/src/github.com/libgit2/git2go \
 && git checkout -f next \
 && git submodule update --init \
 && make install \

 && apt-get purge -y cmake pkg-config \
 && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* \

 && go get github.com/splitsh/lite \
 && go build -v -o /usr/local/bin/splitsh-lite github.com/splitsh/lite \

 && rm -rf /go/*

RUN apt-get update \
 && apt-get install -y python-pip \

 && pip install pyyaml \

 && apt-get purge -y python-pip \
 && apt-get --purge autoremove \
 && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

ADD gitsplit /usr/local/bin/gitsplit
ENV PYTHONUNBUFFERED=0

WORKDIR /srv
CMD ["gitsplit"]
