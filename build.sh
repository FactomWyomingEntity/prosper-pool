#!/usr/bin/env bash

go install -ldflags="-X github.com/FactomWyomingEntity/private-pool/config.CompiledInBuild=`git rev-parse HEAD` -X github.com/FactomWyomingEntity/private-pool/config.CompiledInVersion=`git describe --tags`"