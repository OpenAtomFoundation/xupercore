#!/bin/bash

cd `dirname $0`/../../../

HOMEDIR=`pwd`
OUTPUTDIR="$HOMEDIR/output"
INSTALLDIR="$HOMEDIR/testnet"

if [ ! -d "$OUTPUTDIR/bin" ];then
    echo "Please compile first! cmd:make all"
    exit 1
fi

# make install dir
if [ ! -d "$INSTALLDIR" ];then
    mkdir $INSTALLDIR
fi
rm -rf "$INSTALLDIR/*"

function installNode() {
    node=$1

    # build
    if [ ! -d "$INSTALLDIR/$node" ]; then
        mkdir "$INSTALLDIR/$node"
    fi

    cp -r "$OUTPUTDIR/bin" "$INSTALLDIR/$node/bin"
    cp -r "$HOMEDIR/example/xchain/auto/control.sh" "$INSTALLDIR/$node/"
    cp -r "$HOMEDIR/example/xchain/data/mock/$node/conf" "$INSTALLDIR/$node/conf"
    cp -r "$HOMEDIR/example/xchain/data/mock/$node/data" "$INSTALLDIR/$node/data"
    cp -r "$HOMEDIR/example/xchain/data/genesis" "$INSTALLDIR/$node/data/genesis"
    
    echo "finish $node install."
}

# install network
installNode "node1"
installNode "node2"
installNode "node3"

echo "install done!"
