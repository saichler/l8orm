#!/usr/bin/env bash
# Use the protoc image to run protoc.sh and generate the bindings.
docker run -e PROTO=relational.proto --mount type=bind,source="$PWD",target=/home/proto/ -it saichler/protoc:latest

# Now move the generated bindings to the models directory and clean up
mkdir -p ../go/types
mv ./types/*.pb.go ../go/types/.
rm -rf ./types
