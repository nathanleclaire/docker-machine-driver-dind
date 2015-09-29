FROM golang:1.5

ENV REPO github.com/nathanleclaire/docker-machine-dind

RUN go get github.com/aktau/github-release
WORKDIR /go/src/${REPO}
ADD . /go/src/${REPO}
