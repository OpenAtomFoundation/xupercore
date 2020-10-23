#!/bin/bash

protoc -I ./googleapis/ -I ./ --go_out=plugins=grpc:./ ./server.proto
