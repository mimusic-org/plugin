#!/bin/bash

# Install protoc-gen-go if not installed
if ! command -v protoc-gen-go &> /dev/null
then
    echo "Installing protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

# Install protoc-gen-go-plugin if not installed
if ! command -v protoc-gen-go-plugin &> /dev/null
then
    echo "Installing protoc-gen-go-plugin..."
    go install github.com/knqyf263/go-plugin/cmd/protoc-gen-go-plugin@latest
fi

# Generate plugin files
protoc --go-plugin_out=. \
       --go-plugin_opt=paths=source_relative \
       api/pbplugin/plugin.proto

echo "Generation complete!"
