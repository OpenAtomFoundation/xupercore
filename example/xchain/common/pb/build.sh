#!/bin/bash

protoc -I ./ --go_out=plugins=grpc:./ ./xchain.proto
