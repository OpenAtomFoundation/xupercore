#!/bin/bash

cd `dirname $0`/../../../

HOMEDIR=`pwd`
OUTDIR="$HOMEDIR/output"
XVMDIR="$HOMEDIR/.compile_cache/xvm"

# make output dir
if [ ! -d "$OUTDIR" ];then
    mkdir $OUTDIR
fi
rm -rf "$OUTDIR/*"

# check xvm
if [ ! -f "$XVMDIR/wasm2c" ];then
    echo "please first execute: make xvm"
    exit 1
fi

function buildpkg() {
    output=$1
    pkg=$2

    version=`git rev-parse --abbrev-ref HEAD`
    if [ $? != 0 ]; then
        version="unknow"
    fi
    
    commitId=`git rev-parse --short HEAD`
    if [ $? != 0 ]; then
        commitId="unknow"
    fi

    buildTime=$(date "+%Y-%m-%d-%H:%M:%S")
    
    
    # build
    if [ ! -d "$OUTDIR/bin" ]; then
        mkdir "$OUTDIR/bin"
    fi

    ldflags="-X main.Version=$version -X main.BuildTime=$buildTime -X main.CommitID=$commitId"
    echo "go build -o "$OUTDIR/bin/$output" -ldflags \"$ldflags\" $pkg"

    go build -o "$OUTDIR/bin/$output" -ldflags \
        "-X main.Version=$version -X main.BuildTime=$buildTime -X main.CommitID=$commitId" $pkg
}

# build xuperos
buildpkg xchain "$HOMEDIR/example/xchain/cmd/chain/main.go"
buildpkg xchain-cli "$HOMEDIR/example/xchain/cmd/client/main.go"

# build output
cp -r "$HOMEDIR/example/xchain/conf" "$OUTDIR"
cp "$HOMEDIR/example/xchain/auto/control.sh" "$OUTDIR"
mkdir -p "$OUTDIR/data"
cp -r "$HOMEDIR/example/xchain/data/genesis" "$OUTDIR/data"
cp "$XVMDIR/wasm2c" "$OUTDIR/bin"

echo "compile done!"
