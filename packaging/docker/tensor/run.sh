#!/bin/bash

if [ ! -e "/firstrun" ]; then
    go get -u -v ./...
    echo "run" > "/firstrun"
fi

go run tensord/main.go