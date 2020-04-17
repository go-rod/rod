# docker build -t ysmood/rod .

FROM alpine:latest

LABEL src="https://github.com/ysmood/rod/blob/master/lib/Dockerfile"

# to take advantage of the cahce, don't put everything in one line

RUN apk add chromium

# install fonts for pupular languages, so that screenshot for we page can work properly
# https://wiki.alpinelinux.org/wiki/Fonts
# https://en.wikipedia.org/wiki/Languages_used_on_the_Internet
RUN apk add font-noto font-noto-cjk
RUN apk add font-noto-arabic font-noto-thai font-noto-devanagari

# timezone support
RUN apk add tzdata

# run chrome with non-root mode so that we don't have to use --no-sandbox flag
RUN addgroup -S rod && adduser -S rod -G rod
USER rod

WORKDIR /home/rod
