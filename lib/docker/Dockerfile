# To build the image:
#
#     docker build -t ghcr.io/go-rod/rod -f lib/docker/Dockerfile .
#

# build rod-manager
FROM golang AS go

ARG goproxy="https://goproxy.io,direct"

COPY . /rod
WORKDIR /rod
RUN go env -w GO111MODULE=on && go env -w GOPROXY=$goproxy
RUN go build ./lib/launcher/rod-manager
RUN go run ./lib/utils/get-browser
RUN ln -s $(go run ./lib/utils/get-browser) /usr/bin/chrome

FROM ubuntu:bionic

ARG apt_sources="http://archive.ubuntu.com"

RUN sed -i "s|http://archive.ubuntu.com|$apt_sources|g" /etc/apt/sources.list && \
    apt-get update && \
    apt-get install --no-install-recommends -y \
    # chromium dependencies
    libnss3 \
    libxss1 \
    libasound2 \
    libxtst6 \
    libgtk-3-0 \
    libgbm1 \
    ca-certificates \
    # fonts
    fonts-liberation fonts-noto-color-emoji fonts-noto-cjk \
    # timezone
    tzdata \
    # processs reaper
    dumb-init \
    # headful mode support, for example: $ xvfb-run chromium-browser --remote-debugging-port=9222
    xvfb \
    # cleanup
    && rm -rf /var/lib/apt/lists/*

# processs reaper
ENTRYPOINT ["dumb-init", "--"]

COPY --from=go /root/.cache/rod /root/.cache/rod
COPY --from=go /usr/bin/chrome /usr/bin/
COPY --from=go /rod/rod-manager /usr/bin/

CMD rod-manager
