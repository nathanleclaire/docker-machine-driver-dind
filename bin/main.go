package main

import (
	"github.com/docker/machine/libmachine/drivers/plugin"
	"github.com/nathanleclaire/docker-machine-driver-dind"
)

func main() {
	plugin.RegisterDriver(new(dind.Driver))
}
