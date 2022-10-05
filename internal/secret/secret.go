package secret

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"os"
)

const keySize = 2048
const nodePrivateKeyFile = "nodePrivateKey.pem"

type orgPrivKeyMsg struct {
	EncryptedPrivateKey string
}

var nodePrivateKey, orgPrivateKey *rsa.PrivateKey

func InitNodeKeypair() ([]byte, error) {
	pemFile, err := os.Open(nodePrivateKeyFile)
	if err != nil {
		if os.IsNotExist(err) {
			err := generateNodeKeypair()
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		defer pemFile.Close()

		byteValue, err := io.ReadAll(pemFile)
		if err != nil {
			return nil, err
		}

		block, _ := pem.Decode(byteValue)
		if block == nil || block.Type != "RSA PRIVATE KEY" {
			return nil, errors.New("failed to decode PEM block containing private key")
		}

		// add org private key to node
		nodePrivateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
	}

	// return public key
	publicKeyPem := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&nodePrivateKey.PublicKey),
	}
	publicKeyPemBytes := pem.EncodeToMemory(publicKeyPem)
	if publicKeyPemBytes == nil {
		return nil, errors.New("failed to encode PEM block containing public key")
	}

	return publicKeyPemBytes, nil
}

func generateNodeKeypair() error {
	var err error
	nodePrivateKey, err = rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return err
	}

	// dump private key to file
	privateKeyPem := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(nodePrivateKey),
	}

	privateKeyFile, err := os.OpenFile(nodePrivateKeyFile, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	return pem.Encode(privateKeyFile, privateKeyPem)
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

func DecryptEnv(enc string) (string, error) {
	decBytes, err := rsa.DecryptPKCS1v15(rand.Reader, orgPrivateKey, []byte(enc))
	if err != nil {
		return "", err
	}
	return string(decBytes), nil
}
