#!/bin/bash

cd `dirname $0`/../

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
    cp -r "$HOMEDIR/auto/control.sh" "$INSTALLDIR/$node/"
    cp -r "$HOMEDIR/data/mock/$node/conf" "$INSTALLDIR/$node/conf"
    cp -r "$HOMEDIR/data/mock/$node/data" "$INSTALLDIR/$node/data"
    cp -r "$HOMEDIR/data/genesis" "$INSTALLDIR/$node/data/genesis"
    cp -r "$HOMEDIR/conf/engine.yaml" "$INSTALLDIR/$node/conf/engine.yaml"
    cp -r "$HOMEDIR/conf/env.yaml" "$INSTALLDIR/$node/conf/env.yaml"
    cp -r "$HOMEDIR/conf/ledger.yaml" "$INSTALLDIR/$node/conf/ledger.yaml"
    cp -r "$HOMEDIR/conf/log.yaml" "$INSTALLDIR/$node/conf/log.yaml"
    cp -r "$HOMEDIR/conf/contract.yaml" "$INSTALLDIR/$node/conf/contract.yaml"

    echo "finish $node install."
}

# install network
installNode "node1"
installNode "node2"
installNode "node3"

echo "install done!"
