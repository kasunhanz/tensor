#!/bin/bash
set -e

BINDATA_ARGS="-o util/bindata.go -pkg util"

if [ "$1" == "watch" ]; then
	BINDATA_ARGS="-debug ${BINDATA_ARGS}"
	echo "Creating util/bindata.go with file proxy"
else
	echo "Creating util/bindata.go"
fi

if [ "$1" == "ci_test" ]; then
	echo "Creating CI Test config.json"

	cat > config.json <<EOF
{
	"mysql": {
		"host": "127.0.0.1:3306",
		"user": "hilbert",
		"pass": "hilbert",
		"name": "hilbertspace"
	},
	"session_db": "127.0.0.1:6379",
	"port": ":8010"
}
EOF

fi

echo "Adding bindata"

go-bindata $BINDATA_ARGS config.json db/migrations/

if [ "$1" == "ci_test" ]; then
	exit 0
fi

if [ "$1" == "watch" ]; then
	reflex -r '\.go$' -s -d none -- sh -c 'go run cli/main.go'
	exit 0
fi

cd cli
gox -os="linux" -output="hilbertspace_{{.OS}}_{{.Arch}}" ./...
#gox -os="linux darwin windows openbsd" -output="hilbertspace_{{.OS}}_{{.Arch}}" ./...

if [ "$CIRCLE_ARTIFACTS" != "" ]; then
	rsync -a hilbertspace_* $CIRCLE_ARTIFACTS/
	exit 0
fi
