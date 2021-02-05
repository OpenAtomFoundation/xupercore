#!/bin/bash

# path in work root
# pwd
# /path/to/github.com/username/xupercore
# sh kernel/engines/xuperos/xpb/build.sh

protoc -I ../ \
--go_opt=paths=source_relative \
--go_out=plugins=grpc:./ \
-I ./ kernel/engines/xuperos/xpb/xpb.proto
