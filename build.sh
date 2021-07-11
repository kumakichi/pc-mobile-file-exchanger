#!/bin/bash

VERSION=$(git describe --tags --always)
Date=$(date -u +%Y%m%d_%H%M%S)
LDFLAGS="-X main.Version=${VERSION}_${Date}"
go build -v -ldflags "$LDFLAGS"
