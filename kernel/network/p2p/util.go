package p2p

import (
	"crypto/rand"
	defaulttls "crypto/tls"
	defaultx509 "crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"io/ioutil"
	math_rand "math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tls "github.com/tjfoc/gmsm/gmtls"
	"github.com/tjfoc/gmsm/gmtls/gmcredentials"
	"github.com/tjfoc/gmsm/x509"

	iaddr "github.com/ipfs/go-ipfs-addr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"google.golang.org/grpc/credentials"

	"github.com/xuperchain/xupercore/kernel/network/config"
)

// serverName  为key,缓存 creds
var serverNameMap = make(map[string]credentials.TransportCredentials)

//修改全局变量 serverNameMap 加锁
var mu sync.Mutex

func NewTLS(path, serviceName string) (credentials.TransportCredentials, error) {

	if len(serviceName) < 1 {
		return nil, errors.New("serviceName is empty")
	}

	//如果缓存中有值
	if creds, ok := serverNameMap[serviceName]; ok {
		return creds, nil
	}

	mu.Lock()
	defer mu.Unlock()
	//读取 cacert.pem 证书
	bs, err := ioutil.ReadFile(filepath.Join(path, "cacert.pem"))
	if err != nil {
		return nil, err
	}
	cacert, err := ioutil.ReadFile(filepath.Join(path, "cacert.pem"))
	if err != nil {
		return nil, err
	}
	pb, _ := pem.Decode(cacert)
	x509cert, err := x509.ParseCertificate(pb.Bytes)
	if err != nil {
		return nil, err
	}

	if strings.Contains(strings.ToLower(x509cert.SignatureAlgorithm.String()), "sm") { //国密
		certPool := x509.NewCertPool()
		ok := certPool.AppendCertsFromPEM(bs)
		if !ok {
			return nil, err
		}
		certificate, err := tls.LoadX509KeyPair(filepath.Join(path, "cert.pem"), filepath.Join(path, "private.key"))
		if err != nil {
			return nil, err
		}
		creds := gmcredentials.NewTLS(
			&tls.Config{
				GMSupport:    tls.NewGMSupport(),
				ServerName:   serviceName,
				Certificates: []tls.Certificate{certificate, certificate},
				RootCAs:      certPool,
				ClientCAs:    certPool,
				ClientAuth:   tls.RequireAndVerifyClientCert,
			})
		serverNameMap[serviceName] = creds
		return creds, nil
	} else { //非国密
		certPool := defaultx509.NewCertPool()
		ok := certPool.AppendCertsFromPEM(bs)
		if !ok {
			return nil, err
		}

		certificate, err := defaulttls.LoadX509KeyPair(filepath.Join(path, "cert.pem"), filepath.Join(path, "private.key"))
		if err != nil {
			return nil, err
		}

		creds := credentials.NewTLS(
			&defaulttls.Config{
				ServerName:   serviceName,
				Certificates: []defaulttls.Certificate{certificate},
				RootCAs:      certPool,
				ClientCAs:    certPool,
				ClientAuth:   defaulttls.RequireAndVerifyClientCert,
			})
		serverNameMap[serviceName] = creds
		return creds, nil
	}

}

// GenerateKeyPairWithPath generate xuper net key pair
func GenerateKeyPairWithPath(path string) error {
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		return err
	}

	if len(path) == 0 {
		path = config.DefaultNetKeyPath
	}

	privData, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return err
	}

	if err = os.MkdirAll(path, 0777); err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(path, "net_private.key"), []byte(base64.StdEncoding.EncodeToString(privData)), 0700)
}

// GetPeerIDFromPath return peer id of given private key path
func GetPeerIDFromPath(path string) (string, error) {
	pk, err := GetKeyPairFromPath(path)
	if err != nil {
		return "", err
	}

	pid, err := peer.IDFromPrivateKey(pk)
	if err != nil {
		return "", err
	}
	return pid.Pretty(), nil
}

// GetKeyPairFromPath get xuper net key from file path
func GetKeyPairFromPath(path string) (crypto.PrivKey, error) {
	if len(path) == 0 {
		path = config.DefaultNetKeyPath
	}

	data, err := ioutil.ReadFile(filepath.Join(path, "net_private.key"))
	if err != nil {
		return nil, err
	}

	privData, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, err
	}
	return crypto.UnmarshalPrivateKey(privData)
}

// GetPemKeyPairFromPath get xuper pem private key from file path
func GetPemKeyPairFromPath(path string) (crypto.PrivKey, error) {
	if len(path) == 0 {
		path = config.DefaultNetKeyPath
	}

	keyFile, err := ioutil.ReadFile(filepath.Join(path, "private.key"))
	if err != nil {
		return nil, err
	}

	keyBlock, _ := pem.Decode(keyFile)
	return crypto.UnmarshalRsaPrivateKey(keyBlock.Bytes)
}

// GeneratePemKeyFromNetKey get pem format private key from net private key
func GeneratePemKeyFromNetKey(path string) error {
	privKey, err := GetKeyPairFromPath(path)
	if err != nil {
		return err
	}

	bytes, err := privKey.Raw()
	if err != nil {
		return err
	}

	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: bytes,
	}
	return ioutil.WriteFile(filepath.Join(path, "private.key"), pem.EncodeToMemory(block), 0700)
}

// GenerateNetKeyFromPemKey get net private key from pem format private key
func GenerateNetKeyFromPemKey(path string) error {
	priv, err := GetPemKeyPairFromPath(path)
	if err != nil {
		return err
	}

	privData, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return err
	}

	if err = os.MkdirAll(path, 0777); err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(path, "net_private.key"), []byte(base64.StdEncoding.EncodeToString(privData)), 0700)
}

// GenerateUniqueRandList get a random unique number list
func GenerateUniqueRandList(size int, max int) []int {
	r := math_rand.New(math_rand.NewSource(time.Now().UnixNano()))
	if max <= 0 || size <= 0 {
		return nil
	}
	if size > max {
		size = max
	}
	randList := r.Perm(max)
	return randList[:size]
}

// GetPeerIDByAddress return peer ID corresponding to peerAddr
func GetPeerIDByAddress(peerAddr string) (peer.ID, error) {
	addr, err := iaddr.ParseString(peerAddr)
	if err != nil {
		return "", err
	}
	peerinfo, err := peer.AddrInfoFromP2pAddr(addr.Multiaddr())
	if err != nil {
		return "", err
	}
	return peerinfo.ID, nil
}
