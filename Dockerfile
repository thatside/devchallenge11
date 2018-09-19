FROM golang:1.10

ENV GOPATH /opt/go:$GOPATH
ENV PATH /opt/go/bin:$PATH
ADD . /opt/go/src/main/
WORKDIR /opt/go/src/main

RUN go get github.com/mafredri/cdp golang.org/x/sync/errgroup && \
    go build main.go
CMD ["./main"]