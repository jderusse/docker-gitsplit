FROM python:2-slim

ENV SPLITSH_VERSION=1.0.1

RUN apt-get update \
 && apt-get install -y curl \

 && curl -L https://github.com/splitsh/lite/releases/download/v${SPLITSH_VERSION}/lite_linux_amd64.tar.gz|tar zxf - && mv ./splitsh-lite /bin \

 && apt-get purge --auto-remove -y curl \
 && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

RUN echo "deb http://ftp.debian.org/debian jessie-backports main" > /etc/apt/sources.list.d/backports.list \
 && apt-get update \
 && apt-get install -y -t jessie-backports \
        git \

 && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

RUN pip install pyyaml

ADD gitsplit /usr/local/bin/gitsplit
ENV PYTHONUNBUFFERED=0

WORKDIR /srv
CMD ["gitsplit"]
