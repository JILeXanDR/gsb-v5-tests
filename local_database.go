package main

import (
	"fmt"
	"log"
	"sync"

	"gsb-v5-tests/proto"
)

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

type localDatabase struct {
	lists []localList

	lock *sync.RWMutex
}

func newLocalDatabase() *localDatabase {
	return &localDatabase{
		lists: make([]localList, 0),
		lock:  &sync.RWMutex{},
	}
}

func (d *localDatabase) findThreatsByHashes(hashes []uint32) (threatTypes []proto.ThreatType, err error) {
	d.lock.RLock()
	defer d.lock.RUnlock()

	for _, list := range d.lists {
		// log.Printf("check url %s (expressions=%d) in list %s", rawURL, len(expressions), list.name)
		found, err := searchLocalListHashes(list, hashes)
		if err != nil {
			return threatTypes, err
		}
		if found {
			if len(list.threatTypes) == 0 {
				return nil, fmt.Errorf("list does not have assigned threat types")
			}
			threatTypes = append(threatTypes, list.threatTypes...)
			break
		}
	}

	return threatTypes, nil
}

func (d *localDatabase) updateListss(result *proto.ListHashListsResponse, hashLists []*proto.HashList) {
	lists, err := buildLocalLists(result, hashLists)
	if err != nil {
		log.Printf("failed to build local lists: %+v", err)
		return
	}

	d.lock.Lock()
	d.lists = lists
	d.lock.Unlock()

	for _, list := range d.lists {
		log.Printf(
			`updated local list "%s", entries=%d, threatTypes=%v, likelySafeTypes=%v, supportedHashLengths=%v, description=%s`,
			list.name,
			len(list.decodedUint32Hashes),
			list.threatTypes,
			list.likelySafeTypes,
			list.supportedHashLengths,
			list.description,
		)
	}
}

func buildLocalLists(result *proto.ListHashListsResponse, hashLists []*proto.HashList) ([]localList, error) {
	var localLists []localList

	for i, list := range result.HashLists {
		hashList := hashLists[i]

		name := hashList.Name

		log.Printf("decoding list hashes=%s", name)

		if res := list.CompressedRemovals; res != nil {
			log.Printf("decode CompressedRemovals (RiceDeltaEncoded32Bit), first=%d entries=%d, rice=%d", res.FirstValue, res.EntriesCount, res.RiceParameter)

			decodedHashes, err := DecodeUint32HashPrefixes(res.EncodedData, uint32(res.EntriesCount), res.FirstValue, uint32(res.RiceParameter))
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

		// TODO: this is list for GC (Global Cache) and does not work.
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

			decodedHashes, err := DecodeUint256HashPrefixes(res.EncodedData, uint32(res.EntriesCount), Uint256{
				Part1: res.FirstValueFirstPart,
				Part2: res.FirstValueSecondPart,
				Part3: res.FirstValueThirdPart,
				Part4: res.FirstValueFourthPart,
			}, uint32(res.RiceParameter))
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
