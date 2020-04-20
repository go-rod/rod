FROM ysmood/rod

RUN apk add git go

CMD go test -v
