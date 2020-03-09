#!/bin/bash

set -e

appname='grabmail'
dockerName='data_grabmail'

VERSION=v`date "+%Y%m%d%H%M"`

git checkout master
git pull

git tag $VERSION
git push origin master --tag

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '-s -w' -o $appname

chmod a+x $appname
docker build -t hub.ifchange.com/data_group/$dockerName:$VERSION .
docker tag hub.ifchange.com/data_group/$dockerName:$VERSION hub.ifchange.com/data_group/$dockerName:latest
docker push hub.ifchange.com/data_group/$dockerName:$VERSION
docker push hub.ifchange.com/data_group/$dockerName:latest
rm $appname
