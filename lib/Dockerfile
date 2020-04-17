# docker build -t ysmood/rod .

FROM alpine:latest

LABEL src="https://github.com/ysmood/rod/blob/master/lib/Dockerfile"

# to take advantage of the cahce, don't put everything in one line

RUN apk update

RUN apk add chromium

# install fonts for English, Chinese, Japanese, Korean
# https://wiki.alpinelinux.org/wiki/Fonts
RUN apk add font-noto font-noto-cjk

# timezone
RUN apk add tzdata

# run chrome with non-root mode so that we don't have to use --no-sandbox flag
RUN addgroup -S rod && adduser -S rod -G rod
USER rod

WORKDIR /home/rod
