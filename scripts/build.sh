#!/bin/bash
set -e -x

cd `dirname $0`/../

# build wasm2c
make -C xvm/compile/wabt -j 4
cp xvm/compile/wabt/build/wasm2c ./

# build framework and tools
function buildpkg() {
    output=$1
    pkg=$2
    buildVersion=`git rev-parse --abbrev-ref HEAD`
    buildDate=$(date "+%Y-%m-%d-%H:%M:%S")
    commitHash=`git rev-parse --short HEAD`
    go build -o $output -ldflags "-X main.buildVersion=$buildVersion -X main.buildDate=$buildDate -X main.commitHash=$commitHash" $pkg
}

buildpkg xchain-cli github.com/xuperchain/xupercore/cmd/cli
buildpkg xchain github.com/xuperchain/xupercore/cmd/xchain
buildpkg xdev github.com/xuperchain/xupercore/cmd/xdev
buildpkg dump_chain github.com/xuperchain/xupercore/test
buildpkg relayer github.com/xuperchain/xupercore/cmd/relayer/relayer

# build plugins
echo "OS:"${PLATFORM}
echo "## Build Plugins..."
mkdir -p plugins/kv plugins/crypto plugins/consensus plugins/contract
go build --buildmode=plugin --tags single -o plugins/kv/kv-ldb-single.so.1.0.0 github.com/xuperchain/xupercore/kv/kvdb/plugin-ldb
go build --buildmode=plugin -o plugins/crypto/crypto-default.so.1.0.0 github.com/xuperchain/xupercore/crypto/client/xchain/plugin_impl
go build --buildmode=plugin -o plugins/crypto/crypto-schnorr.so.1.0.0 github.com/xuperchain/xupercore/crypto/client/schnorr/plugin_impl
go build --buildmode=plugin -o plugins/consensus/consensus-single.so.1.0.0 github.com/xuperchain/xupercore/consensus/single
go build --buildmode=plugin -o plugins/consensus/consensus-tdpos.so.1.0.0 github.com/xuperchain/xupercore/consensus/tdpos/main
go build --buildmode=plugin -o plugins/p2p/p2p-p2pv1.so.1.0.0 github.com/xuperchain/xupercore/p2p/p2pv1/plugin_impl
go build --buildmode=plugin -o plugins/p2p/p2p-p2pv2.so.1.0.0 github.com/xuperchain/xupercore/p2p/p2pv2/plugin_impl
go build --buildmode=plugin -o plugins/xendorser/xendorser-default.so.1.0.0 github.com/xuperchain/xupercore/server/xendorser/plugin-default
go build --buildmode=plugin -o plugins/xendorser/xendorser-proxy.so.1.0.0 github.com/xuperchain/xupercore/server/xendorser/plugin-proxy

# build output dir
mkdir -p output
output_dir=output
mv xchain-cli xchain ${output_dir}
mv wasm2c ${output_dir}
mv dump_chain ${output_dir}
mv xdev ${output_dir}
mv relayer ${output_dir}
cp -rf plugins ${output_dir}
cp -rf data ${output_dir}
cp -rf conf ${output_dir}
cp -rf cmd/relayer/conf/relayer.yaml ${output_dir}/conf
cp -rf cmd/quick_shell/* ${output_dir}
mkdir -p ${output_dir}/data/blockchain
