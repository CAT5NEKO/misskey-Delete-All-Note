package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

type Client struct {
	Token    string
	Host     string
	Endpoint string
	HTTP     *http.Client
}

func NewClient() (*Client, error) {
	_ = godotenv.Load()

	token := os.Getenv("TOKEN")
	host := os.Getenv("HOST")
	if token == "" || host == "" {
		return nil, fmt.Errorf("TOKEN or HOST not set in .env")
	}

	return &Client{
		Token:    token,
		Host:     host,
		Endpoint: fmt.Sprintf("https://%s/api/", host),
		HTTP:     &http.Client{},
	}, nil
}

func (c *Client) Post(api string, args map[string]interface{}, result interface{}) error {
	args["i"] = c.Token
	body, err := json.Marshal(args)
	if err != nil {
		return err
	}

	resp, err := c.HTTP.Post(c.Endpoint+api, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		fmt.Printf("Warning: HTTP %d returned from %s\n", resp.StatusCode, api)
		if result != nil {
			_ = json.NewDecoder(resp.Body).Decode(result)
		}
		return nil
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}
