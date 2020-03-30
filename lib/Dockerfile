FROM alpine:latest

LABEL src="https://github.com/ysmood/rod/blob/master/lib/Dockerfile"

# to take advantage of the cahce, don't put everything in one line

RUN apk update

RUN apk add chromium

# libs for font
RUN apk add harfbuzz
RUN apk add freetype
RUN apk add ttf-freefont

# libs for secure networking
RUN apk add nss
RUN apk add ca-certificates

# timezone
RUN apk add tzdata

# run chrome with non-root mode so that we don't have to use --no-sandbox flag
RUN addgroup -S rod && adduser -S rod -G rod
USER rod

WORKDIR /home/rod
