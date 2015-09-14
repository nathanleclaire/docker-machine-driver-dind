default: build

build:
	./make.sh

clean:
	docker rmi dindbase

run:
	docker run -d -P --privileged nathanleclaire/docker-machine-dind
