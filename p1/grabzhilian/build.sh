#!/bin/bash

set -e

git checkout master
git pull

CGO_ENABLED=0 go build -a -v -ldflags '-s -w'
upx -9 grabzhilian
chmod a+x grabzhilian
mv grabzhilian grabzhilian_linux_amd64
sz grabzhilian_linux_amd64

