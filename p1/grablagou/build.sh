#!/bin/bash

set -e

git checkout master
git pull

CGO_ENABLED=0 go build -a -v -ldflags '-s -w'
upx -9 grablagou
chmod a+x grablagou
mv grablagou grablagou_linux_amd64
sz grablagou_linux_amd64

