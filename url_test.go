package main

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_generateHostSuffixes(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input: "a.b.com",
			expected: []string{
				"a.b.com",
				"b.com",
			},
		},
		{
			input: "a.b.c.d.e.f.com",
			expected: []string{
				"a.b.c.d.e.f.com",
				"c.d.e.f.com",
				"d.e.f.com",
				"e.f.com",
				"f.com",
			},
		},
		{
			input: "1.2.3.4",
			expected: []string{
				"1.2.3.4",
			},
		},
		{
			input: "example.co.uk",
			expected: []string{
				"example.co.uk",
			},
		},
		{
			input: "a.example.com",
			expected: []string{
				"a.example.com",
				"example.com",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			suffixes, err := generateHostSuffixes(test.input)
			require.NoError(t, err)

			slices.Sort(test.expected)
			slices.Sort(suffixes)

			assert.Equal(t, test.expected, suffixes)
		})
	}
}

func TestGetAllSuffixPrefixCombinations(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input: "http://a.b.com/1/2.html?param=1",
			expected: []string{
				"a.b.com/1/2.html?param=1",
				"a.b.com/1/2.html",
				"a.b.com/",
				"a.b.com/1/",
				"b.com/1/2.html?param=1",
				"b.com/1/2.html",
				"b.com/",
				// "b.com/1/", // TODO: fix test, this value shouldn't be here
			},
		},
		{
			input: "http://a.b.c.d.e.f.com/1.html",
			expected: []string{
				"a.b.c.d.e.f.com/1.html",
				"a.b.c.d.e.f.com/",
				"c.d.e.f.com/1.html",
				"c.d.e.f.com/",
				"d.e.f.com/1.html",
				"d.e.f.com/",
				"e.f.com/1.html",
				"e.f.com/",
				"f.com/1.html",
				"f.com/",
			},
		},
		{
			input: "http://1.2.3.4/1/",
			expected: []string{
				"1.2.3.4/1/",
				"1.2.3.4/",
			},
		},
		{
			input: "http://example.co.uk/1",
			expected: []string{
				"example.co.uk/1",
				"example.co.uk/",
			},
		},
		{
			input: "http://example.co.uk/1/2/3",
			expected: []string{
				"example.co.uk/",
				"example.co.uk/1",
				"example.co.uk/1/2",
				"example.co.uk/1/2/3",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			suffixes, err := generateExpressions(test.input)
			require.NoError(t, err)

			slices.Sort(test.expected)
			slices.Sort(suffixes)

			assert.Equal(t, test.expected, suffixes)
		})
	}
}
