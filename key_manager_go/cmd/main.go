package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"syscall/js"
)

const PEM_PRIVATE_KEY = "RSA PRIVATE KEY"
const PEM_PUBLIC_KEY = "RSA PUBLIC KEY"

func main() {
	js.Global().Set("genKey",js.FuncOf(GenerateKeyPair))
	<-make(chan struct{})
}

func GenerateKeyPair(js.Value,[]js.Value) interface{} {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err.Error()
	}
	block := &pem.Block{
		Type:  PEM_PRIVATE_KEY,
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	}
	privateKey := string(pem.EncodeToMemory(block))
	pubBlock := &pem.Block{
		Type:  PEM_PUBLIC_KEY,
		Bytes: x509.MarshalPKCS1PublicKey(&privKey.PublicKey),
	}
	_ =string(pem.EncodeToMemory(pubBlock))
	return privateKey
}