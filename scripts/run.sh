#!/usr/bin/env bash
GITROOT=`git rev-parse --show-toplevel`
PATTERN=${1:-01-weather}

export GOPATH="$GITROOT/gocode"
mkdir -p $GOPATH

LIBPATH=$GOPATH/src/github.com/kurrik/witgo

mkdir -p $LIBPATH
rm -rf $LIBPATH/v1
ln -s $GITROOT/v1 $LIBPATH/v1

echo "Getting dependencies"
go get github.com/kurrik/oauth1a
go get github.com/kurrik/twittergo

shift
NAME=`ls -d $GITROOT/examples/* | grep $PATTERN | head -n1`
echo "Running example '$NAME' with args '$@'"
go run $NAME/*.go $@
