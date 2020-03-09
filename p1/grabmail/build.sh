#!/bin/bash

git checkout master
git pull

CGO_ENABLED=0 go build -a -v -ldflags '-s -w'
upx -9 grabmail
chmod a+x grabmail
mv grabmail grabmail_linux_amd64
sz grabmail_linux_amd64

