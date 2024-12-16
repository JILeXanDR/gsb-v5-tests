package main

import (
	"fmt"
	"math"
	"net"
	"net/url"
	"strings"

	"golang.org/x/net/publicsuffix"
)

func generateExpressions(rawURL string) ([]string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %v", err)
	}

	// Ensure the URL has a path
	if parsedURL.Path == "" {
		parsedURL.Path = "/"
	}

	// Generate host suffixes
	hostSuffixes, err := generateHostSuffixes(parsedURL.Hostname())
	if err != nil {
		return nil, err
	}

	// Generate path prefixes
	pathPrefixes, err := generatePathPrefixes(parsedURL.Path, parsedURL.RawQuery)
	if err != nil {
		return nil, err
	}

	return combinePrefixesAndSuffixes(hostSuffixes, pathPrefixes), nil
}

func combinePrefixesAndSuffixes(hosts []string, paths []string) []string {
	var combinations []string
	for _, host := range hosts {
		for _, path := range paths {
			combinations = append(combinations, host+path)
		}
	}
	return combinations
}

func canonicalizeHostname(hostname string) (string, error) {
	// Remove leading and trailing dots
	hostname = strings.Trim(hostname, ".")

	// Replace consecutive dots with a single dot
	hostname = strings.ReplaceAll(hostname, "..", ".")

	// Check if it's an IP address
	if ip := net.ParseIP(hostname); ip != nil {
		if ip.To4() != nil {
			// IPv4 address
			return ip.String(), nil
		}
	}

	// Use EffectiveTLDPlusOne to get the base domain
	eTLDPlusOne, err := publicsuffix.EffectiveTLDPlusOne(hostname)
	if err != nil {
		return hostname, err // Return the original hostname if eTLD+1 extraction fails
	}

	return eTLDPlusOne, nil
}

func generateHostSuffixes(originalHostname string) ([]string, error) {
	// Canonicalize the hostname
	baseDomain, err := canonicalizeHostname(originalHostname)
	if err != nil {
		return nil, err
	}

	// Collect all host suffixes starting from the original hostname down to the base domain
	parts := strings.Split(originalHostname, ".")
	baseParts := strings.Split(baseDomain, ".")
	var suffixes []string

	for i := 0; i <= len(parts)-len(baseParts); i++ {
		if i != 0 && len(parts[i:]) > 5 {
			continue
		}

		suffix := strings.Join(parts[i:], ".")
		suffixes = append(suffixes, suffix)
	}

	return suffixes, nil
}

func generatePathPrefixes(path string, query string) ([]string, error) {
	var prefixes []string

	// Add the full path with query string if present
	if query != "" {
		prefixes = append(prefixes, path+"?"+query)
	}

	// Add the full path without query string
	prefixes = append(prefixes, path)

	// Add the root path
	if path != "/" {
		prefixes = append(prefixes, "/")
	}

	// Add intermediate path prefixes
	parts := strings.Split(strings.Trim(path, "/"), "/")

	limit := int(math.Min(float64(len(parts)), 4))

	for i := 1; i < limit; i++ {
		prefix := "/" + strings.Join(parts[:i], "/") + "/"
		if prefix != path { // Prevent re-adding the full path
			prefixes = append(prefixes, prefix)
		}
	}

	return prefixes, nil
}
