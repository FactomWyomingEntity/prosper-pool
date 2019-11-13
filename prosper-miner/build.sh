#!/usr/bin/env bash

gox -os="linux darwin windows freebsd" -arch="amd64" -ldflags="-X github.com/FactomWyomingEntity/prosper-pool/config.CompiledInBuild=`git rev-parse HEAD` -X github.com/FactomWyomingEntity/prosper-pool/config.CompiledInVersion=`git describe --tags`"
gox -os="linux" -arch="arm arm64" -ldflags="-X github.com/FactomWyomingEntity/prosper-pool/config.CompiledInBuild=`git rev-parse HEAD` -X github.com/FactomWyomingEntity/prosper-pool/config.CompiledInVersion=`git describe --tags`"

./rename.sh