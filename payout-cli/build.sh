#!/usr/bin/env bash

gox -os="linux darwin windows" -arch="amd64" -ldflags="-X github.com/FactomWyomingEntity/prosper-pool/config.CompiledInBuild=`git rev-parse HEAD` -X github.com/FactomWyomingEntity/prosper-pool/config.CompiledInVersion=`git describe --tags`"


