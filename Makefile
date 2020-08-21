ifeq ($(OS),Windows_NT)
  PLATFORM="Windows"
else
  ifeq ($(shell uname),Darwin)
    PLATFORM="MacOS"
  else
    PLATFORM="Linux"
  endif
endif

all: build 
export GO111MODULE=on
export GOFLAGS=-mod=vendor
XCHAIN_ROOT := ${PWD}
export XCHAIN_ROOT
PATH := ${PWD}/xvm/compile/wabt/build:$(PATH)

build:
	PLATFORM=$(PLATFORM) ./scripts/build.sh

test:
	go test -coverprofile=coverage.txt -covermode=atomic ./...
	# test wasm sdk
	GOOS=js GOARCH=wasm go build github.com/xuperchain/xupercore/contractsdk/go/driver

contractsdk:
	make -C contractsdk/cpp build
	make -C contractsdk/cpp test

clean:
	rm -rf output
	rm -rf logs
	rm -rf plugins
	rm -f xchain-cli
	rm -f xchain
	rm -f dump_chain
	rm -f event_client

.PHONY: all test clean
