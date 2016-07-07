#!/bin/bash
set -e

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

if [ "$1" == "ci_test" ]; then
	exit 0
fi

if [ "$1" == "watch" ]; then
	reflex -r '\.go$' -s -d none -- sh -c 'go run cli/main.go'
	exit 0
fi

cd cli
gox -os="linux" -arch="amd64 386" -cgo -verbose -output="hilbertspace_{{.OS}}_{{.Arch}}" ./...