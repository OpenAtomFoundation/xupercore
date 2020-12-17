#!/bin/bash

# protoc v3.7.1
# protoc-gen-go v1.3.3

protoc -I ./ ./network.proto --go_opt=paths=source_relative --go_out=plugins=grpc:./
protoc -I ./ ./permission.proto --go_opt=paths=source_relative --go_out=plugins=grpc:./
protoc -I ./ ./contract.proto --go_opt=paths=source_relative --go_out=plugins=grpc:./
protoc -I ./ ./ledger.proto --go_opt=paths=source_relative --go_out=plugins=grpc:./
