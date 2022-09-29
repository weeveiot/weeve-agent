package secret

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
)

const keySize = 2048

type orgPrivKeyMsg struct {
	EncryptedPrivateKey string
}

func GenerateNodeKeypair() ([]byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, err
	}
	publicKey := &privateKey.PublicKey

	// dump private key to file
	privateKeyPem := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	privateKeyFile, err := os.OpenFile("nodePrivateKey.pem", os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}
	err = pem.Encode(privateKeyFile, privateKeyPem)
	if err != nil {
		return nil, err
	}

	// dump public key to file
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
	// add org key

	return nil
}
