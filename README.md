# Usage

To build and install, make sure you have a Go 1.6 environment and Docker Machine
greater than or equal to 0.5.1. Currently, a `godep restore` is also needed if
the deps do not resolve correctly for you.

Then, just run:

```
$ make && make install
```

You need to also ensure that you have built and run the image which gets used to
spawn the `dind` containers:

```
$ cd image && make && cd -
```

(Ideally this should be pulled automatically, but this hasn't been added to the
driver yet; I would be happy to accept such a change as a patch).

After that, configure your Docker client to talk to a daemon instance (e.g.
`eval $(docker-machine env)`) and run the `create`:

```
$ docker-machine create -d dind my-first-dind
```

Creation is pretty speedy because waiting for the machines to boot is usually
the most time-consuming part of the `create` process, but has definitely
highlighted some existing issues with the provisioning and ideally could be
faster.

Some features, such as Swarm support, will not work properly unless you are
creating with `--dind-host` set to `unix:///var/run/docker.sock` (the default on
Linux).  This is because if `--dind-host` is set to use the socket, the `ip`
command will return the container's IP on the `docker0`, and consequently these
operations will work similarly to how operating on a public IPv4 address does in
the standard Machine cloud provider model.  If `--dind-host` is a remote host
(e.g. boot2docker), the Docker URL provided for connection will be based on an
arbitrary high port NATed from the container to the host's IP address. 
