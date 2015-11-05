#!/bin/bash

set -e

if [[ -z "$HUBUSER" ]]; then
    export HUBUSER=nathanleclaire
fi

if [[ $(docker images -q dindbase | wc -l) -eq 0 ]]; then
    echo 'FROM ubuntu:14.04
RUN apt-get update && apt-get install -y curl && rm -r /var/lib/apt/lists/*
RUN curl https://get.docker.com/ >/bootstrap.sh
RUN chmod +x /bootstrap.sh' | docker build -t dindbootstrap -
    # Install Docker... needs privileged mode
    docker run -it --privileged dindbootstrap sh -c '/bootstrap.sh && rm -r /var/lib/apt/lists/* -vf'
    docker commit $(docker ps -lq) dindbase
fi

docker build -t ${HUBUSER}/docker-machine-dind .
echo '*************************************************************************************
* FINISHED BUILDING THE DOCKER IN DOCKER IMAGE - nathanleclaire/docker-machine-dind *
*************************************************************************************'
