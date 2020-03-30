# the ysmood/rod has the chromium we need to run the test
FROM ysmood/rod

USER root
RUN apk add git go
USER rod

WORKDIR /home/rod

COPY . .

RUN go test -c

CMD go test -v
