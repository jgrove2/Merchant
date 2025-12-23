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

	"backend/internal/kalshi/types"
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

	log.Println(pathWithoutQuery)

	sig, err := c.signMessage(method, pathWithoutQuery, timestamp)
	if err != nil {
		return nil, fmt.Errorf("signing error: %w", err)
	}

	// 3. Create Request
	url := c.BaseURL + path

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// 4. Set Headers
	req.Header.Set("KALSHI-ACCESS-KEY", c.AccessKey)
	req.Header.Set("KALSHI-ACCESS-SIGNATURE", sig)
	req.Header.Set("KALSHI-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("Content-Type", "application/json")

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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("kalshi api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// signMessage implements the RSA-PSS signing logic required by Kalshi
func (c *Client) signMessage(method, path, timestamp string) (string, error) {
	// Message format: timestamp + method + path
	msg := timestamp + method + path

	hashed := sha256.Sum256([]byte(msg))

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

type BalanceResponse struct {
	Balance int64 `json:"balance"`
}

type APIResponse struct {
	Markets []types.MarketData `json:"markets"`
	Cursor  string             `json:"cursor"`
}

type MarketsResponse struct {
	Markets []types.SimplifiedMarket `json:"markets"`
	Cursor  string                   `json:"cursor"`
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

func (c *Client) GetMarkets(limit int, cursor string, status string) (*MarketsResponse, error) {
	// Build query parameters
	path := "/trade-api/v2/markets"
	params := []string{}

	if limit > 0 {
		params = append(params, fmt.Sprintf("limit=%d", limit))
	}

	if cursor != "" {
		params = append(params, fmt.Sprintf("cursor=%s", cursor))
	}

	if status != "" {
		params = append(params, fmt.Sprintf("status=%s", status))
	}

	if len(params) > 0 {
		path = path + "?" + strings.Join(params, "&")
	}

	log.Printf("[Kalshi] Fetching markets with path: %s", path)
	data, err := c.DoRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var fullResponse APIResponse
	if err := json.Unmarshal(data, &fullResponse); err != nil {
		return nil, err
	}

	simplified := make([]types.SimplifiedMarket, len(fullResponse.Markets))

	for i, m := range fullResponse.Markets {
		simplified[i] = types.SimplifiedMarket{
			Ticker:      m.Ticker,
			EventTicker: m.EventTicker,
			Title:       m.Title,
			YesAsk:      m.YesAsk,
			NoAsk:       m.NoAsk,
			YesSubTitle: m.YesSubTitle,
			NoSubTitle:  m.NoSubTitle,
			Status:      m.Status,
		}
	}

	return &MarketsResponse{
		Markets: simplified,
		Cursor:  fullResponse.Cursor,
	}, nil
}
