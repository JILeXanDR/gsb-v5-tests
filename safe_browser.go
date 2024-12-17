package main

import (
	"context"
	"time"

	"gsb-v5-tests/proto"
)

type CheckResult struct {
	Safe    bool
	Threats []proto.ThreatType
}

type SafeBrowserOption func(*safeBrowserOptions)

type safeBrowserOptions struct {
	key string
	api api
}

func WithAPIKey(key string) SafeBrowserOption {
	return func(options *safeBrowserOptions) {
		options.key = key
	}
}

func WithAPIClient(api api) SafeBrowserOption {
	return func(options *safeBrowserOptions) {
		options.api = api
	}
}

type SafeBrowser struct {
	localDatabase *localDatabase
}

func NewSafeBrowser(options ...SafeBrowserOption) (*SafeBrowser, error) {
	opts := new(safeBrowserOptions)

	for _, option := range options {
		option(opts)
	}

	var api api

	if opts.api != nil {
		api = opts.api
	} else {
		client, err := newAPIClient(opts.key)
		if err != nil {
			return nil, err
		}
		api = client
	}

	sb := &SafeBrowser{
		localDatabase: newLocalDatabase(api),
	}

	tctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sb.localDatabase.update(tctx); err != nil {
		return nil, err
	}

	return sb, nil
}

func (sb *SafeBrowser) Run(ctx context.Context) {
	sb.localDatabase.runSelfUpdates(ctx)
}

func (sb *SafeBrowser) CheckURLs(ctx context.Context, urls []string) ([]CheckResult, error) {
	var results []CheckResult

	for _, url := range urls {
		threats, err := sb.getURLThreats(url)
		if err != nil {
			return nil, err
		}
		results = append(results, CheckResult{
			Safe:    len(threats) == 0,
			Threats: threats,
		})
	}

	return results, nil
}

func (sb *SafeBrowser) getURLThreats(rawURL string) ([]proto.ThreatType, error) {
	expressions, err := generateExpressions(rawURL)
	if err != nil {
		return nil, err
	}

	hashesUint32 := make([]uint32, len(expressions))
	// hashesUint256 := make([]Uint256, len(expressions))

	for i, expression := range expressions {
		hashesUint32[i] = hashUint32FourBytes(expression)
		// hashesUint256[i] = hashUint256(expression)
	}

	// if _, err := sb.localDatabase.findLikelySafeByHashes(hashesUint256); err != nil {
	// 	return nil, err
	// }

	threatTypes, err := sb.localDatabase.findThreatsByHashes(hashesUint32)
	if err != nil {
		return nil, err
	}

	return threatTypes, nil
}
