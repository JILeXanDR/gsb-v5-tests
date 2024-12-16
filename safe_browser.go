package main

import (
	"context"
	"log"
	"slices"
	"sync"
	"time"

	"gsb-v5-tests/proto"
)

type CheckResult struct {
	Safe bool
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
	api           api
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
		api:           api,
		localDatabase: newLocalDatabase(),
	}

	tctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sb.updateLocalDatabase(tctx); err != nil {
		return nil, err
	}

	return sb, nil
}

func (sb *SafeBrowser) RunUpdates(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := sb.updateLocalDatabase(ctx); err != nil {
				log.Printf("updating failed: %+v", err)
			}
		}
	}
}

func (sb *SafeBrowser) CheckURLs(ctx context.Context, urls []string) ([]CheckResult, error) {
	var results []CheckResult

	for _, url := range urls {
		safe, err := sb.checkURLIsSafe(url)
		if err != nil {
			return nil, err
		}
		results = append(results, CheckResult{Safe: safe})
	}

	return results, nil
}

func (sb *SafeBrowser) updateLocalDatabase(ctx context.Context) error {
	log.Printf("running local database updates...")

	loadHashLists := sync.OnceValue(func() []*proto.HashList {
		log.Printf("loading list names once...")

		hashListsResult, _, err := sb.api.v5alpha1HashLists(ctx)
		if err != nil {
			return nil
		}

		log.Printf("got hash lists, count=%d", len(hashListsResult.HashLists))

		return hashListsResult.HashLists
	})

	hashLists := loadHashLists()

	var listNames []string

	for _, list := range hashLists {
		listNames = append(listNames, list.Name)
	}

	result, _, err := sb.api.v5alpha1HashListsBatchGet(ctx, listNames)
	if err != nil {
		return err
	}

	sb.localDatabase.updateListss(result, hashLists)

	// TODO: there are 2 lists: srf, mwb without description and threat types.

	return nil
}

func (sb *SafeBrowser) checkURLIsSafe(rawURL string) (bool, error) {
	threats, err := sb.getURLThreats(rawURL)
	if err != nil {
		return false, err
	}

	return len(threats) == 0, nil
}

func (sb *SafeBrowser) getURLThreats(rawURL string) ([]proto.ThreatType, error) {
	expressions, err := generateExpressions(rawURL)
	if err != nil {
		return nil, err
	}

	hashes := make([]uint32, len(expressions))

	for i, expression := range expressions {
		hashes[i] = hashPrefix(expression)
	}

	threatTypes, err := sb.localDatabase.findThreatsByHashes(hashes)
	if err != nil {
		return nil, err
	}

	return threatTypes, nil
}

func searchLocalListHashes(list localList, hashes []uint32) (bool, error) {
	for _, hash := range hashes {
		index, found := slices.BinarySearch(list.decodedUint32Hashes, hash)
		log.Printf("check hash %d in %s list: found=%v", hash, list.name, found)
		if found {
			log.Printf("hash found: index=%d, threats=%v", index, list.threatTypes)
			return true, nil
		}
	}

	log.Printf("hash not found in the list %s", list.name)

	return false, nil
}
