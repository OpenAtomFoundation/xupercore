#!/bin/bash

# path in work root
# pwd
# /path/to/github.com/username/xupercore
# sh protos/build.sh

# protoc v3.7.1
# protoc-gen-go v1.3.3

protoc -I ../ \
--go_opt=paths=source_relative \
--go_out=plugins=grpc:./ \
-I ./ protos/network.proto

protoc -I ../ \
--go_opt=paths=source_relative \
--go_out=plugins=grpc:./ \
-I ./ protos/permission.proto

protoc -I ../ \
--go_opt=paths=source_relative \
--go_out=plugins=grpc:./ \
-I ./ protos/contract.proto

protoc -I ../ \
--go_opt=paths=source_relative \
--go_out=plugins=grpc:./ \
-I ./ protos/ledger.proto

protoc -I ../ \
--go_opt=paths=source_relative \
--go_out=plugins=grpc:./ \
-I ./ protos/event.proto

protoc -I ../ \
--go_opt=paths=source_relative \
--go_out=plugins=grpc:./ \
-I ./ protos/proposal.proto
