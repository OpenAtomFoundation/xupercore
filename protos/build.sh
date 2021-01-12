#!/bin/bash

# protoc v3.7.1
# protoc-gen-go v1.3.3

protoc -I ./ -I ../../ network.proto --go_opt=paths=source_relative --go_out=plugins=grpc:./
protoc -I ./ -I ../../ permission.proto --go_opt=paths=source_relative --go_out=plugins=grpc:./
protoc -I ./ -I ../../ contract.proto --go_opt=paths=source_relative --go_out=plugins=grpc:./
protoc -I ./ -I ../../ ledger.proto --go_opt=paths=source_relative --go_out=plugins=grpc:./
