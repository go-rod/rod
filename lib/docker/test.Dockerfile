# this is only for rod unit tests

FROM rodorg/rod

RUN apk add git go

CMD go test -v -run Test
