package main

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_localDBChecks(t *testing.T) {
	err := godotenv.Load()
	require.NoError(t, err)

	apiKey := os.Getenv("GSB_API_KEY")
	require.NotEmpty(t, apiKey, "GSB_API_KEY variable is not set")

	log.Printf("use API key: %s", apiKey)

	api := mustNewAPIClient(apiKey)

	localLists, err := downloadLocalLists(context.TODO(), api, "./testdata")
	require.NoError(t, err)

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
		// {
		// 	input:  "https://testsafebrowsing.appspot.com/s/malware_in_iframe.html",
		// 	isSafe: false,
		// },
		{
			input:  "https://testsafebrowsing.appspot.com/s/unwanted.html",
			isSafe: false,
		},
		// {
		// 	input:  "https://testsafebrowsing.appspot.com/s/trick_to_bill.html",
		// 	isSafe: false,
		// },
		{
			input:  "https://005d975e0e.news-xnifepo.cc",
			isSafe: false,
		},
		{
			input:  "https://example.com",
			isSafe: true,
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			isSafe, err := checkURLIsSafe(test.input, localLists)
			require.NoError(t, err)
			assert.Equal(t, test.isSafe, isSafe)
		})
	}
}
