#!/usr/bin/env bash
set -e

# Use the protoc image to run protoc.sh and generate the bindings.
docker run --user "$(id -u):$(id -g)" -e PROTO=orms.proto --mount type=bind,source="$PWD",target=/home/proto/ -it saichler/protoc:latest

# Now move the generated bindings to the models directory and clean up
rm -rf ../go/types
mkdir -p ../go/types
mv ./types/* ../go/types/.
rm -rf ./types
