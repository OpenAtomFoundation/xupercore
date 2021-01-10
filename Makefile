# init project PATH
HOMEDIR := $(shell pwd)
OUTDIR  := $(HOMEDIR)/output

# init command params
export GO111MODULE=on
X_ROOT_PATH := $(HOMEDIR)
export X_ROOT_PATH

# make, make all
all: clean compile

# make compile, go build
compile: xchain
xchain:
	bash $(HOMEDIR)/example/xchain/auto/build.sh

# make test, test your code
test:
	go test -coverprofile=coverage.txt -covermode=atomic ./...

# make clean
clean:
	rm -rf output

# avoid filename conflict and speed up build
.PHONY: all compile test clean
