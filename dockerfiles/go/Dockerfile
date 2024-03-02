FROM alpine

ARG DAGGER_VERSION=0.10.0
ARG GO_VERSION=1.21.7

RUN apk add --no-cache \
    bash \
    curl \
    docker-cli \
    nodejs

RUN adduser cacidy -D
RUN addgroup docker && addgroup cacidy docker
RUN mkdir /data && chown -R cacidy /data

RUN curl https://cli.configu.com/install.sh | sh

USER cacidy

RUN curl -L https://dl.dagger.io/dagger/install.sh | DAGGER_VERSION=$DAGGER_VERSION BIN_DIR=$HOME/.local/bin sh

RUN curl -Lo /tmp/go.tar.gz "https://go.dev/dl/go$GO_VERSION.linux-amd64.tar.gz" && \
    cd /tmp/ && tar xf go.tar.gz && \
    mv go /home/cacidy/.local/go && \
    rm /tmp/go.tar.gz

ENV PATH=$PATH:/home/cacidy/.local/bin/:/home/cacidy/.local/go/bin

COPY ./cacidy /home/cacidy/.local/bin/cacidy