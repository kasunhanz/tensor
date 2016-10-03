#!/bin/bash

if [ ! -e "/firstrun" ]; then
    go get -u -v ./...
    echo "run" > "/firstrun"
#remove ssh-key from the container
    rm -f /root/.ssh/bitbucket
fi
#hack for permission issue
rm -f /opt/tensor/bin/tensord
go build -v -o /opt/tensor/bin/tensord ./tensord/...
tensord