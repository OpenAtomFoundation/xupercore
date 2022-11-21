module github.com/xuperchain/xupercore

go 1.14

require (
	github.com/ChainSafe/go-schnorrkel v0.0.0-20200626160457-b38283118816 // indirect
	github.com/aws/aws-sdk-go v1.32.4
	github.com/btcsuite/btcutil v1.0.2
	github.com/dgraph-io/badger/v3 v3.2103.1
	github.com/docker/go-connections v0.4.1-0.20180821093606-97c2040d34df // indirect
	github.com/docker/go-units v0.4.0
	github.com/emirpasic/gods v1.12.1-0.20201118132343-79df803e554c
	github.com/fsouza/go-dockerclient v1.6.0
	github.com/gammazero/deque v0.1.0
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.5.2
	github.com/golang/snappy v0.0.3
	github.com/google/gofuzz v1.1.1-0.20200604201612-c04b05f3adfa // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.2
	github.com/hashicorp/golang-lru v0.5.4
	github.com/hyperledger/burrow v0.30.5
	github.com/ipfs/go-ipfs-addr v0.0.1
	github.com/libp2p/go-libp2p v0.14.4
	github.com/libp2p/go-libp2p-circuit v0.4.0
	github.com/libp2p/go-libp2p-core v0.8.6
	github.com/libp2p/go-libp2p-kad-dht v0.15.0
	github.com/libp2p/go-libp2p-kbucket v0.4.7
	github.com/libp2p/go-libp2p-record v0.1.3
	github.com/libp2p/go-libp2p-secio v0.2.2
	github.com/libp2p/go-libp2p-swarm v0.5.3
	github.com/mitchellh/mapstructure v1.1.2
	github.com/multiformats/go-multiaddr v0.3.3
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/prometheus/client_golang v1.10.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.6.2
	github.com/syndtr/goleveldb v1.0.1-0.20200815110645-5c35d600f0ca
	github.com/xuperchain/crypto v0.0.0-20211221122406-302ac826ac90
	github.com/xuperchain/log15 v0.0.0-20190620081506-bc88a9198230
	github.com/xuperchain/xvm v0.0.0-20210126142521-68fd016c56d7
	golang.org/x/crypto v0.0.0-20210506145944-38f3c27a63bf
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	google.golang.org/grpc v1.35.0
)

replace github.com/hyperledger/burrow => github.com/xuperchain/burrow v0.30.6-0.20211229032028-fbee6a05ab0f
