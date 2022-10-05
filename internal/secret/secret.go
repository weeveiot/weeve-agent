package secret

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"os"
)

const keySize = 2048

type orgPrivKeyMsg struct {
	EncryptedPrivateKey string
}

var nodePrivateKey, orgPrivateKey *rsa.PrivateKey

func GenerateNodeKeypair() ([]byte, error) {
	var err error
	nodePrivateKey, err = rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, err
	}
	publicKey := &nodePrivateKey.PublicKey

	// dump private key to file
	privateKeyPem := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(nodePrivateKey),
	}

	privateKeyFile, err := os.OpenFile("nodePrivateKey.pem", os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	err = pem.Encode(privateKeyFile, privateKeyPem)
	if err != nil {
		return nil, err
	}

	// return public key
	publicKeyPem := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(publicKey),
	}
	publicKeyPemBytes := pem.EncodeToMemory(publicKeyPem)
	if publicKeyPemBytes == nil {
		return nil, err
	}

	return publicKeyPemBytes, nil
}

func ProcessOrgPrivKeyMessage(payload []byte) error {
	var orgPrivKeyMessage orgPrivKeyMsg
	err := json.Unmarshal(payload, &orgPrivKeyMessage)
	if err != nil {
		return err
	}

	// decrypt org key
	decryptedOrgPrivateKey, err := rsa.DecryptPKCS1v15(rand.Reader, nodePrivateKey, []byte(orgPrivKeyMessage.EncryptedPrivateKey))
	if err != nil {
		return err
	}

	block, _ := pem.Decode(decryptedOrgPrivateKey)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return errors.New("failed to decode PEM block containing private key")
	}

	// add org private key to node
	orgPrivateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return err
	}

	return nil
}
