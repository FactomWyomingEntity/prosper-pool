#!/usr/bin/env bash

go install -ldflags="-X github.com/FactomWyomingEntity/prosper-pool/config.CompiledInBuild=`git rev-parse HEAD` -X github.com/FactomWyomingEntity/prosper-pool/config.CompiledInVersion=`git describe --tags`"