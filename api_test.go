package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

func Test_apiMethods(t *testing.T) {
	t.Skip()

	require.NoError(t, godotenv.Load())

	apiKey := os.Getenv("GSB_API_KEY")
	require.NotEmpty(t, apiKey, "GSB_API_KEY variable is not set")

	api, err := newAPIClient(apiKey)
	require.NoError(t, err)

	t.Run("v5alpha1HashLists", func(t *testing.T) {
		result, body, err := api.v5alpha1HashLists(context.TODO())
		require.NoError(t, err)
		require.NotEmpty(t, result)
		require.NotEmpty(t, body)

		writeFile(t, "./testdata/hashLists.bin", body)
	})

	t.Run("v5alpha1HashListsBatchGet", func(t *testing.T) {
		result, body, err := api.v5alpha1HashListsBatchGet(context.TODO(), []string{"gc", "se", "mw", "uws", "uwsa", "pha"})
		require.NoError(t, err)
		require.NotEmpty(t, result)
		require.NotEmpty(t, body)

		writeFile(t, "./testdata/hashLists:batchGet.bin", body)
	})
}

func writeFile(t *testing.T, name string, data []byte) {
	t.Helper()

	f, err := os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o777)
	require.NoError(t, err)
	defer f.Close()

	_, err = io.Copy(f, bytes.NewReader(data))
	require.NoError(t, err)
}
