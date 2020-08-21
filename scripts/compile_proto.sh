#!/bin/bash
set -e -x

cd `dirname $0`/../

# install protoc 3.7.1 
# export GO111MODULES=on
# go install github.com/golang/protobuf/protoc-gen-go
# go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway

protoc -I pb pb/*.proto \
	-I pb/googleapis \
	--go_out=plugins=grpc:pb \
	--grpc-gateway_out=logtostderr=true:pb 

protoc -I p2p/pb p2p/pb/*.proto  --go_out=plugins=grpc:p2p/pb

protoc -I xmodel/pb xmodel/pb/versioned_data.proto --go_out=xmodel/pb 

protoc -I contractsdk/pb contractsdk/pb/contract_service.proto \
       --go_out=plugins=grpc,paths=source_relative:contractsdk/go/pbrpc
protoc -I contractsdk/pb contractsdk/pb/contract.proto \
       --go_out=paths=source_relative:contractsdk/go/pb
protoc -I cmd/relayer/pb cmd/relayer/pb/relayer.proto \
       --go_out=cmd/relayer/pb