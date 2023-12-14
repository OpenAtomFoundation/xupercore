#!/bin/bash

# path in work root
# pwd
# /path/to/github.com/username/xupercore
# sh bcs/ledger/xledger/xldgpb/build.sh

protoc -I ../ \
--go_opt=paths=source_relative \
--go_out=plugins=grpc:./ \
-I ./ bcs/ledger/xledger/xldgpb/xledger.proto
