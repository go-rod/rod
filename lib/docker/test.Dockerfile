# this is only for rod unit tests

FROM rodorg/rod

RUN apt-get install -y git curl

# install nodejs
RUN curl -L https://nodejs.org/dist/v15.5.0/node-v15.5.0-linux-x64.tar.xz > node.tar.xz
RUN tar -xf node.tar.xz
RUN mv node-v15.5.0-linux-x64 /root/.node
ENV PATH="/root/.node/bin:${PATH}"
RUN rm node.tar.xz

# install golang
RUN curl -L https://golang.org/dl/go1.15.6.linux-amd64.tar.gz > golang.tar.gz
RUN tar -xf golang.tar.gz
RUN mv go /root/.go
ENV PATH="/root/.go/bin:${PATH}"
ENV CGO_ENABLED=0
RUN rm golang.tar.gz

WORKDIR /t
