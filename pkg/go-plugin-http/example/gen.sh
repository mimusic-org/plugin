#!/bin/bash

# go-plugin-http example 代码生成脚本
# 用于从 proto 文件生成 Go 代码

set -e

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

# Generate proto files
protoc --go-plugin_out=. \
       --go-plugin_opt=paths=source_relative \
       proto/greeter.proto

echo "Generation complete!"
