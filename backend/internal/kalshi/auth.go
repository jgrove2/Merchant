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
	"log"
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

	// Try PKCS8 (standard for modern keys)
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Fallback to PKCS1
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse RSA private key: %v", err)
		}
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not an RSA private key")
	}

	return rsaKey, nil

}

// SignMessage creates the KALSHI-ACCESS-SIGNATURE using RSA-PSS
func (a *AuthCredentials) SignMessage(method, path, timestamp string) (string, error) {
	msg := timestamp + method + path

	hashed := sha256.Sum256([]byte(msg))

	opts := &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
		Hash:       crypto.SHA256,
	}

	// Sign the hashed message
	// Pass crypto.SHA256 as the hash parameter to indicate what hash was used
	signature, err := rsa.SignPSS(rand.Reader, a.PrivateKey, crypto.SHA256, hashed[:], opts)
	if err != nil {
		log.Printf("[Kalshi] Signing error: %v", err)
		return "", err
	}

	sigBase64 := base64.StdEncoding.EncodeToString(signature)

	return sigBase64, nil
}
