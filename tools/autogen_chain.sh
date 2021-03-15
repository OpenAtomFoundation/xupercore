#!/bin/bash

if [ $# -ne 2 ]; then
    echo "param error.example:sh ./tools/autogen_chain.sh hello /home/rd/gopath/src"
    exit 1;
fi

if [ "$1" == "" ] || [ "$2" == "" ]; then
    echo "chain name or output dir unset"
    exit 1;
fi

cd `dirname $0`/../

HOMEDIR=`pwd`
AUTOGENPKG="$HOMEDIR/.autogen"
NEWCHAINNAME="$1"
OUTPUTDIR="$2"

function autogenpkg() {
    echo "start auto pkg..."

    if [ -d "$AUTOGENPKG" ]; then
        rm -rf "$AUTOGENPKG";
    fi
	mkdir -p "$AUTOGENPKG";

    cp -r "$HOMEDIR/example/xchain/hack" "$AUTOGENPKG/auto"
    cp -r "$HOMEDIR/example/xchain/cmd" "$AUTOGENPKG/"
    cp -r "$HOMEDIR/example/xchain/common" "$AUTOGENPKG/"
    cp -r "$HOMEDIR/example/xchain/conf" "$AUTOGENPKG/"
    cp -r "$HOMEDIR/example/xchain/data" "$AUTOGENPKG/"
    cp -r "$HOMEDIR/example/xchain/models" "$AUTOGENPKG/"
    cp -r "$HOMEDIR/example/xchain/service" "$AUTOGENPKG/"
    mv "$AUTOGENPKG/auto/README.md" "$AUTOGENPKG/"
    mv "$AUTOGENPKG/auto/Makefile" "$AUTOGENPKG/"
    mv "$AUTOGENPKG/auto/LICENSE" "$AUTOGENPKG/"
    mv "$AUTOGENPKG/auto/go.mod" "$AUTOGENPKG/"

    if [ -d "$HOMEDIR/.compile_cache" ]; then
        cp -r "$HOMEDIR/.compile_cache" "$AUTOGENPKG/"
    fi

    echo "auto pkg done"
}

function replace() {
    echo "start replace..."

    if [ ! -d "$AUTOGENPKG" ]; then
        echo "please generate code package first"
        exit 1
    fi

    cd "$AUTOGENPKG"
    sedXchainDir="common/def conf auto"
    for dir in $sedXchainDir
    do
        for file in `grep xchain -l -r -n ./$dir`
        do
            #echo "sed -i 's/xchain/$NEWCHAINNAME/g' $file"
            sed -i "s/xchain/$NEWCHAINNAME/g" $file
            if [ $? -ne 0 ]; then
                echo "replace file failed.file:$file"
                exit 1
            fi
        done
    done

    srcStr="github.com\/xuperchain\/xupercore\/example\/xchain\/"
    targetStr="github.com\/xuperchain\/$NEWCHAINNAME\/"
    for f in `grep $srcStr -l -r -n ./`
    do
        #echo "sed -i 's/$srcStr/$targetStr/g' $f"
        sed -i "s/$srcStr/$targetStr/g" $f
        if [ $? -ne 0 ]; then
            echo "replace file failed.file:$file"
            exit 1
        fi
    done

    sed -i "s/github.com\/xuperchain\/xchain/github.com\/xuperchain\/$NEWCHAINNAME/g" ./go.mod
    sed -i "s/rootChain: xuper/rootChain: $NEWCHAINNAME/g" ./conf/engine.yaml 
    sed -i "s/DefChainName = \"xuper\"/DefChainName = \"$NEWCHAINNAME\"/g" ./common/def/def.go

    cd "$HOMEDIR"
    echo "replace done"
}

# auto generate code pkg
autogenpkg

# replace code
replace

# move to output dir
rm -rf "$OUTPUTDIR/github.com/xuperchain/$NEWCHAINNAME/"
mkdir -p "$OUTPUTDIR/github.com/xuperchain/"
cp -r $AUTOGENPKG "$OUTPUTDIR/github.com/xuperchain/$NEWCHAINNAME/"

echo "auto generate chain code succ"
