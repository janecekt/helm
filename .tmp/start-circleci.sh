#!/bin/ash -x

docker container stop go-build-env

docker container rm go-build-env

docker run -it ${ARGS} \
  --rm \
	--net host \
	--name go-build-env \
	-v "$(pwd)/../../../../:/build" \
	circleci/golang:1.16
