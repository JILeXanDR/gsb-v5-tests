package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"google.golang.org/protobuf/proto"
	codegen "gsb-v5-tests/proto"
)

const (
	basePath = "https://safebrowsing.googleapis.com"
)

type api interface {
	v5alpha1HashLists(ctx context.Context) (*codegen.ListHashListsResponse, []byte, error)
	v5alpha1HashListsBatchGet(ctx context.Context, names []string) (*codegen.ListHashListsResponse, []byte, error)
}

type apiClient struct {
	key string
}

func newAPIClient(key string) (*apiClient, error) {
	if key == "" {
		return nil, errors.New("API key is not set")
	}

	return &apiClient{
		key: key,
	}, nil
}

// GET https://safebrowsing.googleapis.com/v5alpha1/hashLists
func (c *apiClient) v5alpha1HashLists(ctx context.Context) (*codegen.ListHashListsResponse, []byte, error) {
	var response codegen.ListHashListsResponse

	body, err := c.request(ctx, "v5alpha1/hashLists", nil, &response)
	if err != nil {
		return nil, nil, err
	}

	return &response, body, nil
}

// GET https://safebrowsing.googleapis.com/v5alpha1/hashLists:batchGet
func (c *apiClient) v5alpha1HashListsBatchGet(ctx context.Context, names []string) (*codegen.ListHashListsResponse, []byte, error) {
	query := url.Values{}

	for _, name := range names {
		query.Add("names", name)
	}

	var response codegen.ListHashListsResponse

	body, err := c.request(ctx, "v5alpha1/hashLists:batchGet", query, &response)
	if err != nil {
		return nil, nil, err
	}

	return &response, body, nil
}

func (c *apiClient) request(ctx context.Context, path string, query url.Values, result proto.Message) ([]byte, error) {
	if query == nil {
		query = url.Values{}
	}

	query.Set("key", c.key)

	rawURL := fmt.Sprintf("%s/%s?%s", basePath, path, query.Encode())

	log.Printf("send request: %s", rawURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed request: status=%d, body=%s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	log.Printf("request result length: %d", len(body))

	unmarshaler := proto.UnmarshalOptions{}

	if err := unmarshaler.Unmarshal(body, result); err != nil {
		return nil, err
	}

	return body, nil
}
