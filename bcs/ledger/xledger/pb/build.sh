#!/bin/bash

protoc -I ./ -I ../../../../protos/ --go_opt=paths=source_relative --go_out=plugins=grpc:./ ./xledger.proto

