#!/bin/bash

protoc -I ./ -I ../../../../../ --go_opt=paths=source_relative --go_out=plugins=grpc:./ ./xchain.proto
