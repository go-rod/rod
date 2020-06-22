# this is only for rod unit tests

FROM ysmood/rod

RUN apk add git go

CMD go test -v
