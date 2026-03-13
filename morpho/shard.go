package main

import (
	"hash/fnv"
	"sync"
)

const numShards = 256

type Shard struct {
	mu        sync.RWMutex
	positions map[string]HFparams // key = borrower address
	_         [56]byte            // padding anti cache-line contention
}

type PositionStore struct {
	shards [numShards]Shard
}

func (s *PositionStore) shard(addr string) *Shard {
	h := fnv.New32a()
	h.Write([]byte(addr))
	return &s.shards[h.Sum32()%numShards]
}

func (s *PositionStore) Set(addr string, p HFparams) {
	sh := s.shard(addr)
	sh.mu.Lock()
	sh.positions[addr] = p
	sh.mu.Unlock()
}

func (s *PositionStore) Get(addr string) (HFparams, bool) {
	sh := s.shard(addr)
	sh.mu.RLock()
	p, ok := sh.positions[addr]
	sh.mu.RUnlock()
	return p, ok
}
