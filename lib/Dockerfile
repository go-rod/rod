# build rod-launcher tool
FROM golang:alpine AS go
COPY . /rod
WORKDIR /rod
RUN go build ./lib/launcher/cmd/rod-launcher


FROM alpine:latest

LABEL src="https://github.com/ysmood/rod/blob/master/lib/Dockerfile"

# to take advantage of the cahce, don't put everything in one line

RUN apk add chromium

# install fonts for pupular languages, so that screenshot for webpages can work properly
# https://wiki.alpinelinux.org/wiki/Fonts
# https://en.wikipedia.org/wiki/Languages_used_on_the_Internet
RUN apk add ttf-freefont
RUN apk add font-noto-cjk
RUN apk add font-noto-arabic font-noto-thai font-noto-devanagari

# timezone support
RUN apk add tzdata

COPY --from=go /rod/rod-launcher /usr/bin/

CMD rod-launcher