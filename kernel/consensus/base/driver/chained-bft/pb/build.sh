#!/bin/bash

# protoc v3.7.1
# protoc-gen-go v1.3.3

protoc -I ./ \
--go_opt=paths=source_relative \
--go_out=plugins=grpc:./ \
./chainedBFTMsg.proto
