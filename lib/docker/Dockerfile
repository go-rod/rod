# Examples:
# 
#   docker build -t ysmood/rod -f lib/docker/Dockerfile .
#
#   // china mirror
#   docker build --build-arg goproxy=https://goproxy.io,direct --build-arg alpine_mirror=mirrors.tuna.tsinghua.edu.cn \
#     -t ysmood/rod -f lib/docker/Dockerfile .

# build rod-launcher tool
FROM golang:alpine AS go

ARG goproxy=""

COPY . /rod
WORKDIR /rod
RUN go env -w GO111MODULE=on && go env -w GOPROXY=$goproxy
RUN go build ./lib/launcher/rod-launcher


FROM alpine:latest

ARG alpine_mirror="dl-cdn.alpinelinux.org"

# to take advantage of the cahce, don't put everything in one line

RUN sed -i "s/dl-cdn.alpinelinux.org/$alpine_mirror/g" /etc/apk/repositories
RUN cat /etc/apk/repositories
RUN apk update
RUN apk add chromium

# install fonts for pupular languages, so that screenshot for webpages can work properly
# https://wiki.alpinelinux.org/wiki/Fonts
# https://en.wikipedia.org/wiki/Languages_used_on_the_Internet
RUN apk add font-noto
RUN apk add font-noto-cjk
RUN apk add font-noto-arabic font-noto-thai font-noto-devanagari
RUN apk add font-noto-emoji --repository=http://dl-cdn.alpinelinux.org/alpine/edge/testing
COPY lib/docker/fonts-local.conf /etc/fonts/local.conf

# timezone support
RUN apk add tzdata

COPY --from=go /rod/rod-launcher /usr/bin/

CMD rod-launcher

EXPOSE 9222
