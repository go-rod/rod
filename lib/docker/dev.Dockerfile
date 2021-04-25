# this is only for rod unit tests

FROM ghcr.io/go-rod/rod

ARG node="https://nodejs.org/dist/v15.5.0/node-v15.5.0-linux-x64.tar.xz"
ARG golang="https://golang.org/dl/go1.15.6.linux-amd64.tar.gz"
ARG apt_sources="http://archive.ubuntu.com"

RUN sed -i "s|http://archive.ubuntu.com|$apt_sources|g" /etc/apt/sources.list && \
    apt-get update && apt-get install --no-install-recommends -y git curl xz-utils && \
    rm -rf /var/lib/apt/lists/*

# install nodejs
RUN curl -L $node > node.tar.xz && \
    tar -xf node.tar.xz && \
    mv node-* /usr/local/lib/.node && \
    rm node.tar.xz
ENV PATH="/usr/local/lib/.node/bin:${PATH}"

# install golang
RUN curl -L $golang > golang.tar.gz && \
    tar -xf golang.tar.gz && \
    mv go /usr/local/lib/.go && \
    rm golang.tar.gz
ENV PATH="/usr/local/lib/.go/bin:${PATH}"
ENV CGO_ENABLED=0

# setup global git ignore
RUN git config --global core.excludesfile ~/.gitignore_global

WORKDIR /t
