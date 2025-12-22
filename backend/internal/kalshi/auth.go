package kalshi

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// AuthCredentials holds your Kalshi API keys
type AuthCredentials struct {
	AccessKey  string
	PrivateKey *rsa.PrivateKey
}

// LoadPrivateKey reads the .key file and parses the RSA Private Key
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	// Kalshi keys are usually PKCS8 or PKCS1
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Fallback to PKCS1 if PKCS8 fails
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %v", err)
		}
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not an RSA private key")
	}

	return rsaKey, nil
}

// SignMessage creates the KALSHI-ACCESS-SIGNATURE using RSA-PSS
func SignMessage(priv *rsa.PrivateKey, method, path, timestamp string) (string, error) {
	// Message format: timestamp + method + path (no query params)
	msg := timestamp + method + path
	hashed := sha256.Sum256([]byte(msg))

	// PSS Options: Salt length matches hash length (SHA256 = 32 bytes)
	opts := &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
		Hash:       crypto.SHA256,
	}

	signature, err := rsa.SignPSS(rand.Reader, priv, crypto.SHA256, hashed[:], opts)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}
