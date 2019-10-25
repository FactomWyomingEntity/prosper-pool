#!/usr/bin/env bash

gox -os="linux darwin windows freebsd" -arch="amd64"
gox -os="linux" -arch="arm arm64"

./rename.sh