package main

import (
	"context"
	"io"
	"log"
	"os"
	"path"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	proto2 "google.golang.org/protobuf/proto"
	"gsb-v5-tests/proto"
)

type fakeAPI struct {
	dataDir string
}

func (fapi *fakeAPI) v5alpha1HashLists(ctx context.Context) (*proto.ListHashListsResponse, []byte, error) {
	var result proto.ListHashListsResponse
	if err := fapi.loadBinaryDataFromFile("hashLists.bin", &result); err != nil {
		return nil, nil, err
	}
	return &result, nil, nil
}

func (fapi *fakeAPI) v5alpha1HashListsBatchGet(ctx context.Context, names []string) (*proto.ListHashListsResponse, []byte, error) {
	var result proto.ListHashListsResponse
	if err := fapi.loadBinaryDataFromFile("hashLists:batchGet.bin", &result); err != nil {
		return nil, nil, err
	}
	return &result, nil, nil
}

func (fapi *fakeAPI) loadBinaryDataFromFile(file string, v *proto.ListHashListsResponse) error {
	log.Printf("load binary data from file %s", file)
	f, err := os.Open(path.Join(fapi.dataDir, file))
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	if err := proto2.Unmarshal(b, v); err != nil {
		return err
	}

	return nil
}

func TestSafeBrowser_CheckURLs(t *testing.T) {
	require.NoError(t, godotenv.Load())

	apiKey := os.Getenv("GSB_API_KEY")
	require.NotEmpty(t, apiKey, "GSB_API_KEY variable is not set")

	sb, err := NewSafeBrowser(
		WithAPIKey(apiKey),
		WithAPIClient(&fakeAPI{
			dataDir: "./testdata",
		}),
	)
	require.NoError(t, err)

	go sb.Run(context.TODO())

	tests := []struct {
		input  string
		isSafe bool
	}{
		{
			input:  "https://testsafebrowsing.appspot.com/s/phishing.html",
			isSafe: false,
		},
		{
			input:  "https://sub.testsafebrowsing.appspot.com/s/phishing.html",
			isSafe: false,
		},
		{
			input:  "https://sub.testsafebrowsing.appspot.com/s/",
			isSafe: true,
		},
		{
			input:  "https://testsafebrowsing.appspot.com/s/",
			isSafe: true,
		},
		{
			input:  "https://testsafebrowsing.appspot.com/s/malware.html",
			isSafe: false,
		},
		{
			input:  "https://testsafebrowsing.appspot.com/s/unwanted.html",
			isSafe: false,
		},
		{
			input:  "https://005d975e0e.news-xnifepo.cc",
			isSafe: false,
		},
		{
			input:  "https://example.com",
			isSafe: true,
		},
		{
			input:  "https://phdelaware.com/",
			isSafe: false,
		},
		{
			input:  "https://example.com",
			isSafe: true,
		},
		{
			input:  "https://google.com",
			isSafe: true,
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			results, err := sb.CheckURLs(context.TODO(), []string{test.input})
			require.NoError(t, err)
			require.Len(t, results, 1)
			assert.Equal(t, test.isSafe, results[0].Safe)
		})
	}
}
