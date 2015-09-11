#!/bin/bash

service docker start
service ssh start
tail -f /var/log/auth.log -f /var/log/docker.log
