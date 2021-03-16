#!/bin/bash

cd `dirname $0`/../

HOMEDIR=`pwd`
AUTOGENPKG="$HOMEDIR/.autogen"
NEWCHAINNAME=""
OUTPUTDIR=""
GITREPO=""

function outusage() {
    echo -e "autogen_chain.sh is a tool for automatic generation of new chain framework.\n"
    echo -e "Usage:\n"
    echo -e "\tsh ./tools/autogen_chain.sh <command> [arguments]\n"
    echo -e "The commands are:\n"
    echo -e "\t-n\tchain name"
    echo -e "\t-r\tgit repositories"
    echo -e "\t-o\toutput dir"
    echo -e "\n"
    echo -e "Example:sh ./tools/autogen_chain.sh -n hello -r bob -o /home/rd/gopath/src\n"
}

Example="sh ./tools/autogen_chain.sh -n hello -r bob -o /home/rd/gopath/src"
if [ $# -ne 6 ]; then
    outusage
    exit 1;
fi

while getopts "n:r:o:" arg
do
    case $arg in
    n)
        NEWCHAINNAME=$OPTARG
        ;;
    r)
        GITREPO=$OPTARG
        ;;
    o)
        OUTPUTDIR=$OPTARG
        ;;
    ?)
        outusage
        exit 1
        ;;
    esac
done

if [ "$NEWCHAINNAME" == "" ] || [ "$GITREPO" == "" ] || [ "$OUTPUTDIR" == "" ]; then
    outusage
    exit 1;
fi

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
    targetStr="github.com\/$GITREPO\/$NEWCHAINNAME\/"
    for f in `grep $srcStr -l -r -n ./`
    do
        #echo "sed -i 's/$srcStr/$targetStr/g' $f"
        sed -i "s/$srcStr/$targetStr/g" $f
        if [ $? -ne 0 ]; then
            echo "replace file failed.file:$file"
            exit 1
        fi
    done

    sed -i "s/github.com\/xuperchain\/xchain/github.com\/$GITREPO\/$NEWCHAINNAME/g" ./go.mod
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
coderepodir="$OUTPUTDIR/github.com/$GITREPO/$NEWCHAINNAME/"
if [ -d "$coderepodir" ]; then
    echo "output dir has exist,auto gen failed.outdir:$coderepodir"   
    exit 1
fi
mkdir -p "$OUTPUTDIR/github.com/$GITREPO/"
cp -r $AUTOGENPKG "$coderepodir"

echo "auto generate chain code succ.outdir:$coderepodir"
