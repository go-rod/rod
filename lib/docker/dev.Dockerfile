# A docker image for rod development.
# To build the image:
#     docker build -t ghcr.io/go-rod/rod:dev -f lib/docker/dev.Dockerfile .

FROM ghcr.io/go-rod/rod

ARG node="https://nodejs.org/dist/v16.14.2/node-v16.14.2-linux-x64.tar.xz"
ARG golang="https://go.dev/dl/go1.18.linux-amd64.tar.gz"
ARG apt_sources="http://archive.ubuntu.com"

RUN sed -i "s|http://archive.ubuntu.com|$apt_sources|g" /etc/apt/sources.list && \
    apt-get update && apt-get install --no-install-recommends -y git curl xz-utils build-essential && \
    rm -rf /var/lib/apt/lists/*

# install nodejs
RUN curl -L $node > node.tar.xz && \
    tar -xf node.tar.xz && \
    mv node-* /usr/local/lib/.node && \
    rm node.tar.xz

# install golang
RUN curl -L $golang > golang.tar.gz && \
    tar -xf golang.tar.gz && \
    mv go /usr/local/lib/go && \
    rm golang.tar.gz

ENV PATH="/usr/local/lib/.node/bin:/usr/local/lib/go/bin:/root/go/bin/:${PATH}"

ENV GODEBUG="tracebackancestors=1000"

# setup global git ignore
RUN git config --global core.excludesfile ~/.gitignore_global
