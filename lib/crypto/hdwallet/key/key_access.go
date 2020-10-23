/*
* Copyright Baidu Inc. All Rights Reserved.
* Package key 把客户端本地存储盘上的加密后存储的私钥解析出来
* 传入的信息是：对称加密的key（也就是用户的支付密码）、私钥存储地址
 */

package key

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/xuperchain/xupercore/lib/crypto/account"
	"github.com/xuperchain/xupercore/lib/crypto/ecies"
	"github.com/xuperchain/xupercore/lib/crypto/hash"
	"github.com/xuperchain/xupercore/lib/crypto/hdwallet/error"
)

// GetBinaryEcdsaPrivateKeyFromFile parse binary ecdsa private key from file
func GetBinaryEcdsaPrivateKeyFromFile(path string, password string) ([]byte, error) {
	filename := path + "private.key"
	content, err := readFileUsingFilename(filename)
	if err != nil {
		log.Printf("readFileUsingFilename failed, the err is %v", err)
		return nil, err
	}

	// 将aes对称加密的密钥扩展至32字节
	newPassword := hash.DoubleSha256([]byte(password))

	originalContent, err := aesDecrypt(content, newPassword)
	if err != nil {
		log.Printf("Decrypt private key file failed, the err is %v", err)
		return nil, err
	}

	return originalContent, nil
}

// GetBinaryEcdsaPrivateKeyFromString 通过二进制字符串获取真实私钥
func GetBinaryEcdsaPrivateKeyFromString(encryptPrivateKey string, password string) ([]byte, error) {
	// 将aes对称加密的密钥扩展至32字节
	newPassword := hash.DoubleSha256([]byte(password))

	originalContent, err := aesDecrypt([]byte(encryptPrivateKey), newPassword)
	if err != nil {
		log.Printf("Decrypt private key file failed, the err is %v", err)
		return nil, err
	}

	return originalContent, nil
}

// GetEcdsaPrivateKeyFromFile parse ecdsa private key from file
func GetEcdsaPrivateKeyFromFile(path string, password string) (*ecdsa.PrivateKey, error) {
	originalContent, err := GetBinaryEcdsaPrivateKeyFromFile(path, password)
	if err != nil {
		log.Printf("GetBinaryEcdsaPrivateKeyFromFile failed, the err is %v", err)
		return nil, err
	}

	return account.GetEcdsaPrivateKeyFromJSON(originalContent)
}

func aesDecrypt(crypted, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(crypted))

	blockMode.CryptBlocks(origData, crypted)

	return pkcs5UnPadding(origData)
}

func pkcs5UnPadding(origData []byte) ([]byte, error) {
	length := len(origData)
	unpadding := int(origData[length-1])

	if length-unpadding <= 0 {
		// 密码错误时 可能会造成为负数
		return nil, spverror.ErrPwWrong
	}

	return origData[:(length - unpadding)], nil
}

func readFileUsingFilename(filename string) ([]byte, error) {
	//从filename指定的文件中读取数据并返回文件的内容。
	//成功的调用返回的err为nil而非EOF。
	//因为本函数定义为读取整个文件，它不会将读取返回的EOF视为应报告的错误。
	contentStr, err := ioutil.ReadFile(filename)
	if os.IsNotExist(err) {
		log.Printf("File [%v] does not exist", filename)
	}
	if err != nil {
		return nil, err
	}
	content, err := base64.StdEncoding.DecodeString(string(contentStr))
	if err != nil {
		//log.Printf("File [%v] content convert failed", filename)
	}
	return content, err
}

// GetEcdsaPublicKeyFromJSON parse ecdsa public key from json encoded string
func GetEcdsaPublicKeyFromJSON(jsonContent []byte) (*ecdsa.PublicKey, error) {
	publicKey := new(account.ECDSAPublicKey)
	err := json.Unmarshal(jsonContent, publicKey)
	if err != nil {
		return nil, err //json有问题
	}
	if publicKey.Curvname != "P-256" && publicKey.Curvname != "P-256-SN" {
		log.Printf("curve [%v] is not supported yet\n", publicKey.Curvname)
		err = fmt.Errorf("curve [%v] is not supported yet", publicKey.Curvname)
		return nil, err
	}
	newPublicKey := &ecdsa.PublicKey{}
	newPublicKey.Curve = elliptic.P256()
	newPublicKey.X = publicKey.X
	newPublicKey.Y = publicKey.Y
	return newPublicKey, nil
}

// EncryptByKey 加密
func EncryptByKey(info string, key string) (string, error) {
	// 将aes对称加密的密钥扩展至32字节
	newPassword := hash.DoubleSha256([]byte(key))

	// 加密info
	cipherInfo, err := aesEncrypt([]byte(info), newPassword)
	if err != nil {
		return "", err
	}
	return string(cipherInfo), err
}

// DecryptByKey 解密
func DecryptByKey(cipherInfo string, key string) (string, error) {
	// 将aes对称加密的密钥扩展至32字节
	newPassword := hash.DoubleSha256([]byte(key))

	// 解密cipherInfo
	info, err := aesDecrypt([]byte(cipherInfo), newPassword)
	if err != nil {
		return "", err
	}
	return string(info), nil
}

// GetPublicKeyByPrivateKey 通过私钥获取公钥
func GetPublicKeyByPrivateKey(binaryPrivateKey string) (string, error) {
	privatekey, err := account.GetEcdsaPrivateKeyFromJSON([]byte(binaryPrivateKey))
	if err != nil {
		return "", err
	}

	// 补充公钥
	jsonPublicKey, err := account.GetEcdsaPublicKeyJSONFormat(privatekey)
	if err != nil {
		return "", err
	}
	return jsonPublicKey, nil
}

// EciesEncryptByJSONPublicKey 使用字符串公钥进行ecies加密
func EciesEncryptByJSONPublicKey(publicKey string, msg string) (string, error) {
	apiPublicKey, err := GetEcdsaPublicKeyFromJSON([]byte(publicKey))
	if err != nil {
		return "", errors.New("api public key is wrong")
	}
	cipherInfo, err := ecies.Encrypt(apiPublicKey, []byte(msg))
	if err != nil {
		return "", spverror.ErrParam
	}
	return string(cipherInfo), nil
}

// EciesDecryptByJSONPrivateKey 使用字符串私钥进行ecies解密
func EciesDecryptByJSONPrivateKey(privateKey string, cipherInfo string) (string, error) {
	apiPrivateKey, err := account.GetEcdsaPrivateKeyFromJSON([]byte(privateKey))
	if err != nil {
		return "", errors.New("api public key is wrong")
	}
	msg, err := ecies.Decrypt(apiPrivateKey, []byte(cipherInfo))
	if err != nil {
		return "", spverror.ErrParam
	}
	return string(msg), nil
}
