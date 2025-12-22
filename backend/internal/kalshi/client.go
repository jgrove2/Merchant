package kalshi

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client handles communication with the Kalshi API
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	AccessKey  string
	PrivateKey *rsa.PrivateKey
}

// NewClient initializes a new Kalshi API client
func NewClient(baseURL, accessKey, keyPath string) (*Client, error) {
	// Validate required parameters
	if baseURL == "" {
		return nil, errors.New("baseURL is required")
	}
	if accessKey == "" {
		return nil, errors.New("accessKey is required")
	}
	if keyPath == "" {
		return nil, errors.New("keyPath is required")
	}

	// Validate URL has a scheme
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		return nil, fmt.Errorf("baseURL must start with http:// or https://, got: %s", baseURL)
	}

	privKey, err := loadPrivateKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	log.Printf("[Kalshi] Successfully loaded private key, key size: %d bits", privKey.N.BitLen())

	return &Client{
		BaseURL:    strings.TrimSuffix(baseURL, "/"),
		AccessKey:  accessKey,
		PrivateKey: privKey,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// DoRequest performs an authenticated request to Kalshi
func (c *Client) DoRequest(method, path string, body io.Reader) ([]byte, error) {
	// 1. Prepare Timestamp
	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())

	// 2. Generate Signature (Path must be stripped of query params for signing)
	pathWithoutQuery := strings.Split(path, "?")[0]

	// Debug logging
	signMsg := timestamp + method + pathWithoutQuery
	log.Printf("[Kalshi] Signing message: %q", signMsg)
	log.Printf("[Kalshi] Timestamp: %s, Method: %s, Path: %s", timestamp, method, pathWithoutQuery)

	sig, err := c.signMessage(method, pathWithoutQuery, timestamp)
	if err != nil {
		return nil, fmt.Errorf("signing error: %w", err)
	}

	log.Printf("[Kalshi] Signature: %s", sig)

	// 3. Create Request
	url := c.BaseURL + path
	log.Printf("[Kalshi] Full URL: %s", url)

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// 4. Set Headers
	req.Header.Set("KALSHI-ACCESS-KEY", c.AccessKey)
	req.Header.Set("KALSHI-ACCESS-SIGNATURE", sig)
	req.Header.Set("KALSHI-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("Content-Type", "application/json")

	log.Printf("[Kalshi] Access Key: %s", c.AccessKey)

	// 5. Execute
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	log.Printf("[Kalshi] Response Status: %d", resp.StatusCode)
	log.Printf("[Kalshi] Response Body: %s", string(respBody))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("kalshi api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// signMessage implements the RSA-PSS signing logic required by Kalshi
func (c *Client) signMessage(method, path, timestamp string) (string, error) {
	// Message format: timestamp + method + path
	msg := timestamp + method + path

	log.Printf("[Kalshi] Raw message to sign: %q", msg)
	log.Printf("[Kalshi] Message bytes: %v", []byte(msg))

	// Hash the message with SHA256
	hashed := sha256.Sum256([]byte(msg))
	log.Printf("[Kalshi] SHA256 hash: %x", hashed)

	// PSS Options: Salt length matches hash length (SHA256 = 32 bytes)
	// MGF1 is the default mask generation function
	opts := &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
		Hash:       crypto.SHA256,
	}

	// Sign the hashed message
	// Pass crypto.SHA256 as the hash parameter to indicate what hash was used
	signature, err := rsa.SignPSS(rand.Reader, c.PrivateKey, crypto.SHA256, hashed[:], opts)
	if err != nil {
		log.Printf("[Kalshi] Signing error: %v", err)
		return "", err
	}

	sigBase64 := base64.StdEncoding.EncodeToString(signature)
	log.Printf("[Kalshi] Signature length: %d bytes", len(signature))
	log.Printf("[Kalshi] Signature (base64): %s", sigBase64)

	return sigBase64, nil
}

// loadPrivateKey parses a .key PEM file into an RSA Private Key
func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
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

// --- Example Method: GetBalance ---

type BalanceResponse struct {
	Balance int64 `json:"balance"`
}

func (c *Client) GetBalance() (int64, error) {
	data, err := c.DoRequest("GET", "/trade-api/v2/portfolio/balance", nil)
	if err != nil {
		return 0, err
	}

	var res BalanceResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return 0, err
	}

	return res.Balance, nil
}
