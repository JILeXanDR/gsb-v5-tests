package main

import (
	"context"
	"io"
	"log"
	"os"
	"path"

	protobufproto "google.golang.org/protobuf/proto"
	"gsb-v5-tests/proto"
)

type localList struct {
	name          string
	decodedHashes []uint32
}

func downloadLocalLists(ctx context.Context, api *apiClient, dataDir string) ([]localList, error) {
	hashLists, err := getHashListsFromFileCache(ctx, api, path.Join(dataDir, "v5alpha1_hashLists.bin"))
	if err != nil {
		return nil, err
	}

	// gc, mw, mwb, pha, se, srf, uws, uwsa
	var listNames []string

	for _, list := range hashLists.HashLists {
		log.Printf("%s, %s, %s, %s", list.GetName(), list.GetMetadata().GetThreatTypes(), list.GetMetadata().GetLikelySafeTypes(), list.GetMetadata().GetSupportedHashLengths())
		listNames = append(listNames, list.GetName())
	}

	println("")

	result, err := getHashListsBatchGetFromFileCache(ctx, api, listNames, path.Join(dataDir, "v5alpha1_hashLists_batchGet.bin"))
	if err != nil {
		return nil, err
	}

	println("")
	log.Printf("build local lists...\n\n")

	localLists, err := buildLocalLists(result, listNames)
	if err != nil {
		return nil, err
	}

	return localLists, nil
}

func buildLocalLists(result *proto.ListHashListsResponse, listNames []string) ([]localList, error) {
	var localLists []localList

	for i, list := range result.HashLists {
		name := listNames[i]

		log.Printf("decoding list hashes=%s", name)

		if res := list.CompressedRemovals; res != nil {
			log.Printf("decode CompressedRemovals (RiceDeltaEncoded32Bit), first=%d entries=%d, rice=%d", res.FirstValue, res.EntriesCount, res.RiceParameter)

			decodedHashes, err := DecodeHashPrefixes(res.FirstValue, uint32(res.RiceParameter), uint32(res.EntriesCount), res.EncodedData)
			if err != nil {
				return nil, err
			}

			log.Printf("CompressedRemovals (RiceDeltaEncoded32Bit) decoded: %d", len(decodedHashes))

			localLists = append(localLists, localList{
				name:          name,
				decodedHashes: decodedHashes,
			})
		}

		println("")

		// TODO: this is list for GC (Global Cache) and does not work.
		// if res := list.GetAdditionsThirtyTwoBytes(); res != nil {
		// 	log.Printf("try to decode GetAdditionsThirtyTwoBytes, rice=%d", res.RiceParameter)
		//
		// 	if decode, err := GolombDecodeBase64(res.EncodedData, res.FirstValueFirstPart, res.EntriesCount, uint32(res.RiceParameter)); err != nil {
		// 		log.Printf("failed to decode GetAdditionsThirtyTwoBytes: %+v", err)
		// 	} else {
		// 		log.Printf("GetAdditionsThirtyTwoBytes decoded: %d", len(decode))
		// 	}
		// }
	}

	return localLists, nil
}

func getHashListsFromFileCache(ctx context.Context, api *apiClient, localPath string) (*proto.ListHashListsResponse, error) {
	return cacheIntoFile[*proto.ListHashListsResponse](localPath, &proto.ListHashListsResponse{}, func() (*proto.ListHashListsResponse, []byte, error) {
		return api.v5alpha1hashLists(ctx)
	})
}

func getHashListsBatchGetFromFileCache(ctx context.Context, api *apiClient, listNames []string, localPath string) (*proto.ListHashListsResponse, error) {
	return cacheIntoFile[*proto.ListHashListsResponse](localPath, &proto.ListHashListsResponse{}, func() (*proto.ListHashListsResponse, []byte, error) {
		return api.v5alpha1hashListsbatchGet(ctx, listNames)
	})
}

func cacheIntoFile[T protobufproto.Message](localPath string, result T, fn func() (T, []byte, error)) (T, error) {
	f, err := os.Open(localPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("cache not found in %s", localPath)

			data, rawBytes, err := fn()
			if err != nil {
				return result, err
			}

			log.Printf("write cache file to %s", localPath)

			if err := writeFile(localPath, rawBytes); err != nil {
				return result, err
			}

			return data, nil
		}

		return result, err
	}

	defer f.Close()

	log.Printf("cache found in %s", localPath)

	b, err := io.ReadAll(f)
	if err != nil {
		return result, err
	}

	if err := protobufproto.Unmarshal(b, result); err != nil {
		return result, err
	}

	return result, nil
}
