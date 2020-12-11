#!/bin/bash

cd `dirname $0`/../../../

HOMEDIR=`pwd`
OUTDIR="$HOMEDIR/output"

# make output dir
if [ ! -d "$OUTDIR" ];then
    mkdir $OUTDIR
fi
rm -rf "$OUTDIR/*"

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
    
    ldflags="-X version.Version=$version -X version.BuildTime=$buildTime -X version.CommitID=$commitId"
    
    # build
    if [ ! -d "$OUTDIR/bin" ]; then
        mkdir "$OUTDIR/bin"
    fi

    go build -o "$OUTDIR/bin/$output" -ldflags $ldflags $pkg
}

# build xuperos
buildpkg xchain "$HOMEDIR/example/xchain/cmd/xchain/main.go"
buildpkg xchain-cli "$HOMEDIR/example/xchain/cmd/client/main.go"

# build output
cp -r "$HOMEDIR/example/xchain/conf" "$OUTDIR"
cp -r "$OUTDIR/example/xchain/data" "$OUTDIR"
cp "$HOMEDIR/example/xchain/auto/control.sh" "$OUTDIR"

