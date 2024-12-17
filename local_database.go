package main

import (
	"context"
	"log"
	"slices"
	"sync"
	"time"

	"gsb-v5-tests/proto"
)

const (
	updatesInterval = 30 * time.Minute
)

// Hardcode available lists like in docs says https://developers.google.com/safe-browsing/reference#available-lists.
var recommendedLists = []*proto.HashList{
	{
		Name: "gc",
		Metadata: &proto.HashListMetadata{
			LikelySafeTypes: []proto.LikelySafeType{proto.LikelySafeType_GENERAL_BROWSING},
		},
	},
	{
		Name: "se",
		Metadata: &proto.HashListMetadata{
			ThreatTypes: []proto.ThreatType{proto.ThreatType_SOCIAL_ENGINEERING},
		},
	},
	{
		Name: "mw",
		Metadata: &proto.HashListMetadata{
			ThreatTypes: []proto.ThreatType{proto.ThreatType_MALWARE},
		},
	},
	{
		Name: "uws",
		Metadata: &proto.HashListMetadata{
			ThreatTypes: []proto.ThreatType{proto.ThreatType_UNWANTED_SOFTWARE},
		},
	},
	{
		Name: "uwsa",
		Metadata: &proto.HashListMetadata{
			ThreatTypes: []proto.ThreatType{proto.ThreatType_UNWANTED_SOFTWARE},
		},
	},
	{
		Name: "pha",
		Metadata: &proto.HashListMetadata{
			ThreatTypes: []proto.ThreatType{proto.ThreatType_POTENTIALLY_HARMFUL_APPLICATION},
		},
	},
}

type localDatabase struct {
	api api

	lists      []localList
	lastUpdate time.Time

	lock *sync.RWMutex
}

func newLocalDatabase(api api) *localDatabase {
	return &localDatabase{
		api:   api,
		lists: make([]localList, 0),
		lock:  &sync.RWMutex{},
	}
}

func (d *localDatabase) runSelfUpdates(ctx context.Context) {
	ticker := time.NewTicker(updatesInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := d.update(ctx); err != nil {
				log.Printf("updating failed: %+v", err)
			}
		}
	}
}

func (d *localDatabase) update(ctx context.Context) error {
	log.Printf("running local database updates...")

	// loadHashLists := sync.OnceValue(func() []*proto.HashList {
	// 	log.Printf("loading list names once...")
	//
	// 	hashListsResult, _, err := d.api.v5alpha1HashLists(ctx)
	// 	if err != nil {
	// 		return nil
	// 	}
	//
	// 	log.Printf("got hash lists, count=%d", len(hashListsResult.HashLists))
	//
	// 	return hashListsResult.HashLists
	// })
	//
	// hashLists := loadHashLists()

	var listNames []string

	for _, list := range recommendedLists {
		listNames = append(listNames, list.Name)
	}

	result, _, err := d.api.v5alpha1HashListsBatchGet(ctx, listNames)
	if err != nil {
		return err
	}

	return d.updateLists(result, recommendedLists)
}

func (d *localDatabase) findLikelySafeByHashes(hashes []Uint256) (likelySafeTypes []proto.LikelySafeType, err error) {
	d.lock.RLock()
	defer d.lock.RUnlock()

	for _, list := range d.lists {
		if len(list.likelySafeTypes) == 0 {
			continue
		}

		if found := list.findUint256Hashes(hashes); found {
			likelySafeTypes = append(likelySafeTypes, list.likelySafeTypes...)
		}
	}

	return likelySafeTypes, nil
}

func (d *localDatabase) findThreatsByHashes(hashes []uint32) (threatTypes []proto.ThreatType, err error) {
	d.lock.RLock()
	defer d.lock.RUnlock()

	for _, list := range d.lists {
		if len(list.threatTypes) == 0 {
			continue
		}

		if found := list.findUint32Hashes(hashes); found {
			threatTypes = append(threatTypes, list.threatTypes...)
		}
	}

	return threatTypes, nil
}

func (d *localDatabase) updateLists(result *proto.ListHashListsResponse, hashLists []*proto.HashList) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	lists, err := buildLocalLists(result, hashLists)
	if err != nil {
		return err
	}

	d.lists = lists
	d.lastUpdate = time.Now()

	for _, list := range d.lists {
		log.Printf(
			`updated local list "%s", entries=%d, threatTypes=%v, likelySafeTypes=%v, description=%s`,
			list.name,
			max(len(list.decodedUint32Hashes), len(list.decodedUint256Hashes)),
			list.threatTypes,
			list.likelySafeTypes,
			list.description,
		)
	}

	return nil
}

type localList struct {
	name                 string
	description          string
	decodedUint32Hashes  []uint32
	decodedUint256Hashes []Uint256
	entriesCount         int32
	threatTypes          []proto.ThreatType
	likelySafeTypes      []proto.LikelySafeType
	supportedHashLengths []proto.HashLength
	version              []byte
	sha256Checksum       []byte
}

func (l *localList) findUint32Hashes(hashes []uint32) bool {
	for _, hash := range hashes {
		index, found := slices.BinarySearch(l.decodedUint32Hashes, hash)
		log.Printf("check hash %d in %s list: found=%v", hash, l.name, found)
		if found {
			log.Printf("hash found: index=%d, threats=%v", index, l.threatTypes)
			return true
		}
	}

	log.Printf("hash not found in the list %s", l.name)

	return false
}

func (l *localList) findUint256Hashes(hashes []Uint256) bool {
	for _, hash := range hashes {
		index, found := slices.BinarySearchFunc(l.decodedUint256Hashes, hash, func(a, b Uint256) int {
			return a.Compare(b)
		})
		log.Printf("check hash %v in %s list: found=%v", hash, l.name, found)
		if found {
			log.Printf("hash found: index=%d, threats=%v", index, l.threatTypes)
			return true
		}
	}

	log.Printf("hash not found in the list %s", l.name)

	return false
}

func buildLocalLists(result *proto.ListHashListsResponse, hashLists []*proto.HashList) ([]localList, error) {
	var localLists []localList

	for i, list := range result.HashLists {
		hashList := hashLists[i]

		name := hashList.Name

		log.Printf("decoding list hashes=%s", name)

		if res := list.CompressedRemovals; res != nil {
			log.Printf("decode CompressedRemovals (RiceDeltaEncoded32Bit), first=%d entries=%d, rice=%d", res.FirstValue, res.EntriesCount, res.RiceParameter)

			enc := &golomb32BitEncoding{
				FirstValue:    res.FirstValue,
				RiceParameter: uint32(res.RiceParameter),
				EncodedData:   res.EncodedData,
				EntryCount:    uint32(res.EntriesCount),
			}

			decodedHashes, err := enc.Decode()
			if err != nil {
				return nil, err
			}

			log.Printf("CompressedRemovals (RiceDeltaEncoded32Bit) decoded: %d", len(decodedHashes))

			localLists = append(localLists, localList{
				name:                 name,
				description:          hashList.Metadata.Description,
				decodedUint32Hashes:  decodedHashes,
				entriesCount:         res.EntriesCount,
				threatTypes:          hashList.Metadata.ThreatTypes,
				likelySafeTypes:      hashList.Metadata.LikelySafeTypes,
				supportedHashLengths: hashList.Metadata.SupportedHashLengths,
				version:              list.Version,
				sha256Checksum:       list.GetSha256Checksum(),
			})
		}

		if res := list.GetAdditionsThirtyTwoBytes(); res != nil {
			log.Printf(
				"decode GetAdditionsThirtyTwoBytes (RiceDeltaEncoded256Bit), first1=%d first2=%d first3=%d first4=%d entries=%d, rice=%d",
				res.FirstValueFirstPart,
				res.FirstValueSecondPart,
				res.FirstValueThirdPart,
				res.FirstValueFourthPart,
				res.EntriesCount,
				res.RiceParameter,
			)

			enc := &golomb256BitEncoding{
				FirstValuePart1: res.FirstValueFirstPart,
				FirstValuePart2: res.FirstValueSecondPart,
				FirstValuePart3: res.FirstValueThirdPart,
				FirstValuePart4: res.FirstValueFourthPart,
				RiceParameter:   uint32(res.RiceParameter),
				EncodedData:     res.EncodedData,
				EntryCount:      uint32(res.EntriesCount),
			}

			// TODO: this doesn't work

			decodedHashes, err := enc.Decode()
			if err != nil {
				return nil, err
			}

			log.Printf("CompressedRemovals (RiceDeltaEncoded256Bit) decoded: %d", len(decodedHashes))

			localLists = append(localLists, localList{
				name:                 name,
				description:          hashList.Metadata.Description,
				decodedUint256Hashes: decodedHashes,
				entriesCount:         res.EntriesCount,
				threatTypes:          hashList.Metadata.ThreatTypes,
				likelySafeTypes:      hashList.Metadata.LikelySafeTypes,
				supportedHashLengths: hashList.Metadata.SupportedHashLengths,
				version:              list.Version,
				sha256Checksum:       list.GetSha256Checksum(),
			})
		}
	}

	return localLists, nil
}
