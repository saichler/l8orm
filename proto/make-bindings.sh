#!/usr/bin/env bash
set -e

wget https://raw.githubusercontent.com/saichler/l8types/refs/heads/main/proto/api.proto
wget https://raw.githubusercontent.com/saichler/l8types/refs/heads/main/proto/notification.proto

# Use the protoc image to run protoc.sh and generate the bindings.
docker run --user "$(id -u):$(id -g)" -e PROTO=orms.proto --mount type=bind,source="$PWD",target=/home/proto/ -it saichler/protoc:latest

rm -rf notification.proto
rm -rf api.proto
rm -rf *.rs

# Now move the generated bindings to the models directory and clean up
rm -rf ../go/types
mkdir -p ../go/types
mv ./types/* ../go/types/.
rm -rf ./types

find . -name "*.go" -type f -exec sed -i 's|"./types/l8services"|"github.com/saichler/l8types/go/types/l8notify"|g' {} +
find . -name "*.go" -type f -exec sed -i 's|"./types/l8api"|"github.com/saichler/l8types/go/types/l8api"|g' {} +
