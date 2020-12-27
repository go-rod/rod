# Examples:
#
#   docker build -t rodorg/rod -f lib/docker/Dockerfile .
#
#   // use mirrors
#   docker build --build-arg goproxy=https://goproxy.io,direct --build-arg apt_sources=https://mirrors.tuna.tsinghua.edu.cn \
#     -t rodorg/rod -f lib/docker/Dockerfile .

# build rod-launcher tool
FROM golang:alpine AS go

ARG goproxy=""

COPY . /rod
WORKDIR /rod
RUN go env -w GO111MODULE=on && go env -w GOPROXY=$goproxy
RUN go build ./lib/launcher/rod-launcher
RUN go run ./lib/utils/get-browser

FROM ubuntu:latest

ARG apt_sources="http://archive.ubuntu.com"

RUN sed -i "s|http://archive.ubuntu.com|$apt_sources|g" /etc/apt/sources.list

RUN apt-get update

# dependencies for Chromium
RUN TZ="America/New_York" DEBIAN_FRONTEND=noninteractive apt-get install -y \
    gconf-service libasound2 libatk1.0-0 libc6 libcairo2 libcups2 \
    libdbus-1-3 libexpat1 libfontconfig1 libgbm1 libgcc1 libgconf-2-4 libgdk-pixbuf2.0-0 libglib2.0-0 \
    libgtk-3-0 libnspr4 libpango-1.0-0 libpangocairo-1.0-0 libstdc++6 libx11-6 libx11-xcb1 libxcb1 \
    libxcomposite1 libxcursor1 libxdamage1 libxext6 libxfixes3 libxi6 libxrandr2 libxrender1 libxss1 \
    libxtst6 ca-certificates fonts-liberation libappindicator1 libnss3 lsb-release xdg-utils

# extra fonts
RUN apt-get install -y fonts-noto-color-emoji fonts-noto-cjk

# headful mode support, for example: $ xvfb-run chromium-browser --remote-debugging-port=9222
RUN apt-get install -y xvfb

# processs reaper
RUN apt-get install -y dumb-init
ENTRYPOINT ["dumb-init", "--"]

COPY --from=go /tmp/rod /tmp/rod
COPY --from=go /rod/rod-launcher /usr/bin/

CMD rod-launcher

# the default port for rod-launcher
EXPOSE 9222
