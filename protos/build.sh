#!/bin/bash

# protoc v3.7.1
# protoc-gen-go v1.3.3

protoc -I ./ ./network.proto --go_out=plugins=grpc:./
