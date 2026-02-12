package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// syntheticFetcher generates fake blocks for demo/testing. No external RPC calls.
type syntheticFetcher struct {
	chainID   string
	nextBlock uint64
}

func newSyntheticFetcher(chainID string) *syntheticFetcher {
	return &syntheticFetcher{chainID: chainID, nextBlock: 0}
}

func (s *syntheticFetcher) FetchNext(ctx context.Context) (*IngestRecord, error) {
	blockNum := s.nextBlock
	s.nextBlock++
	data, err := json.Marshal(map[string]interface{}{
		"block": blockNum, "chain": s.chainID, "ts": time.Now().Unix(),
	})
	if err != nil {
		return nil, err
	}
	return &IngestRecord{
		IdempotencyKey: fmt.Sprintf("%s-%d", s.chainID, blockNum),
		ChainID:        s.chainID,
		BlockNumber:    blockNum,
		Data:           data,
	}, nil
}
