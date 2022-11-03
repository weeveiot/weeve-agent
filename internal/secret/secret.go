package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"os"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const keySize = 2048
const nodePrivateKeyFile = "nodePrivateKey.pem"

type orgPrivKeyMsg struct {
	EncryptedOrgKey string
}

var nodePrivateKey *rsa.PrivateKey
var decryptor cipher.AEAD

func InitNodeKeypair() ([]byte, error) {
	log.Debug("Initializing node keypair...")

	pemFile, err := os.Open(nodePrivateKeyFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("No node private key found. Generating...")
			err := generateNodeKeypair()
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		log.Info("Node private key found.")
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
	log.Info("Node private key set.")
	log.Info("Generating node public key...")

	pk, err := x509.MarshalPKIXPublicKey(&nodePrivateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	// return public key
	publicKeyPem := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pk,
	}
	publicKeyPemBytes := pem.EncodeToMemory(publicKeyPem)
	if publicKeyPemBytes == nil {
		return nil, errors.New("failed to encode PEM block containing public key")
	}

	log.Debug("Generated node public key:\n", string(publicKeyPemBytes))
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
	log.Debug("Received orga's encrypted private key:\n", orgPrivKeyMessage.EncryptedOrgKey)

	encryptedOrgKey, err := base64.StdEncoding.DecodeString(orgPrivKeyMessage.EncryptedOrgKey)
	if err != nil {
		return err
	}

	label := []byte("orgKey")
	orgSecretKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, nodePrivateKey, encryptedOrgKey, label)
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(orgSecretKey)
	if err != nil {
		return err
	}

	decryptor, err = cipher.NewGCM(block)
	if err != nil {
		return err
	}

	log.Info("Orga's private key set.")
	return nil
}

func DecryptEnv(enc string) (string, error) {
	if decryptor == nil {
		return "", errors.New("don't have org's private key. cannot decrypt")
	}
	encBytes, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	nonce, ciphertext := encBytes[:decryptor.NonceSize()], encBytes[decryptor.NonceSize():]

	plaintext, err := decryptor.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
