package keymanagergo

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
)

const PEM_PRIVATE_KEY = "RSA PRIVATE KEY"
const PEM_PUBLIC_KEY = "RSA PUBLIC KEY"
const OPENSSH_PRIVATE_KEY = "OPENSSH PRIVATE KEY"

func GenerateKeyPair() (privateKey string,publicKey string) {
	privKey, err := rsa.GenerateKey(rand.Reader, 1<<12)
	if err != nil {
		return
	}
	block := &pem.Block{
		Type:  PEM_PRIVATE_KEY,
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	}
	privateKey = string(pem.EncodeToMemory(block))
	pubBlock := &pem.Block{
		Type:  PEM_PUBLIC_KEY,
		Bytes: x509.MarshalPKCS1PublicKey(&privKey.PublicKey),
	}
	publicKey =string(pem.EncodeToMemory(pubBlock))
	return
}