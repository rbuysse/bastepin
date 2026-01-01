all: run

build:
  go build

run *args:
  go run . {{args}}

docker-build:
  docker build -t pb -f pb.dockerfile .

docker-run:
  #!/bin/bash
  docker kill pb &> /dev/null
  docker rm pb &> /dev/null
  docker run \
    -d \
    --rm \
    --name pb \
    -p 3001:3001 \
    pb
  echo -e "Run 'docker kill pb' to remove the running container."

fmt:
  #!/usr/bin/env sh
  if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
    gofmt -d -s -l .
    exit 1
  fi
  printf "\033[92mgofmt Success\033[0m\n"

fix-fmt:
  gofmt -w -s .

test:
  go test

ci:
  just build
  just fmt
