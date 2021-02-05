# this is only for rod unit tests

FROM rodorg/rod

ARG node="https://nodejs.org/dist/v15.5.0/node-v15.5.0-linux-x64.tar.xz"
ARG golang="https://golang.org/dl/go1.15.6.linux-amd64.tar.gz"
ARG apt_sources="http://archive.ubuntu.com"

RUN sed -i "s|http://archive.ubuntu.com|$apt_sources|g" /etc/apt/sources.list && \
    apt-get update && apt-get install --no-install-recommends -y git curl xz-utils

# install nodejs
RUN curl -L $node > node.tar.xz
RUN tar -xf node.tar.xz
RUN mv node-* /root/.node
ENV PATH="/root/.node/bin:${PATH}"
RUN rm node.tar.xz

# install golang
RUN curl -L $golang > golang.tar.gz
RUN tar -xf golang.tar.gz
RUN mv go /root/.go
ENV PATH="/root/.go/bin:${PATH}"
ENV CGO_ENABLED=0
RUN rm golang.tar.gz

# setup global git ignore
RUN git config --global core.excludesfile ~/.gitignore_global

WORKDIR /t
