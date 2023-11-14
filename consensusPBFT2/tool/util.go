package tool

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"os"
)

//生成摘要
func GenerateHash(content []byte) []byte {
	hash := sha256.New()
	hash.Write(content)
	return hash.Sum(nil)
}

// 读取公钥文件
func GetPublicKey(path string) rsa.PublicKey {
	publicFile, _ := ioutil.ReadFile(path)
	block, _ := pem.Decode(publicFile)
	pubInterface, _ := x509.ParsePKIXPublicKey(block.Bytes)
	return *pubInterface.(*rsa.PublicKey)
}

// 读取私钥文件
func GetPrivateKey(path string) rsa.PrivateKey {
	privateFile, _ := ioutil.ReadFile(path)
	block, _ := pem.Decode(privateFile)
	privateKey, _ := x509.ParsePKCS1PrivateKey(block.Bytes)
	//fmt.Print(*privateKey)
	s:= privateKey
	return *s
}

//生成密钥对
//path:./resources/   name:node1
func GenerateKey(path string, name string) {

	//创建私钥
	privateKey, _ := rsa.GenerateKey(rand.Reader, 1024)
	derStream := x509.MarshalPKCS1PrivateKey(privateKey)
	block := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: derStream,
	}
	file, _ := os.Create(path + name + "_private_key.pem")
	_ = pem.Encode(file, block)

	//创建公钥
	pub := &privateKey.PublicKey
	derPkix, _ := x509.MarshalPKIXPublicKey(pub)
	block = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derPkix,
	}
	file, _ = os.Create(path + name + "_public_key.pem")
	_ = pem.Encode(file, block)
}

//对消息的散列值进行数字签名
func GetDigest(msg []byte, privateKey rsa.PrivateKey) []byte {
	//计算散列值
	bytes := GenerateHash(msg)
	//SignPKCS1v15使用RSA PKCS#1 v1.5规定的RSASSA-PKCS1-V1_5-SIGN签名方案计算签名
	signature, err := rsa.SignPKCS1v15(rand.Reader, &privateKey, crypto.SHA256, bytes)
	if err != nil {
		panic(signature)
	}
	return signature
}

//验证数字签名
func VerifyDigest(msg []byte, signature []byte, publicKey rsa.PublicKey) bool {
	//计算消息散列值
	bytes := GenerateHash(msg)
	//验证数字签名
	err := rsa.VerifyPKCS1v15(&publicKey, crypto.SHA256, bytes, signature)
	return err == nil
}
