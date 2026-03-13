package morpho

import (
	"hash/fnv"
	"sync"
)

const numShards = 256

type Shard struct {
	mu        sync.RWMutex
	positions map[string]*BorrowPosition // key = borrower address
	_         [56]byte                   // padding anti cache-line contention
}

type PositionStore struct {
	shards [numShards]Shard
}

func NewPositionStore() *PositionStore {
	s := &PositionStore{}
	for i := range s.shards {
		s.shards[i].positions = make(map[string]*BorrowPosition)
	}
	return s
}

func (s *PositionStore) shard(addr string) *Shard {
	// hash
	h := fnv.New32a()
	h.Write([]byte(addr))
	return &s.shards[h.Sum32()%numShards]
}

func (s *PositionStore) Set(addr string, p *BorrowPosition) {
	sh := s.shard(addr)
	sh.mu.Lock()
	sh.positions[addr] = p
	sh.mu.Unlock()
}

func (s *PositionStore) Get(addr string) (*BorrowPosition, bool) {
	sh := s.shard(addr)
	sh.mu.RLock()
	p, ok := sh.positions[addr]
	sh.mu.RUnlock()
	return p, ok
}

func (s *PositionStore) ForEach(fn func(addr string, p *BorrowPosition)) {
	var wg sync.WaitGroup
	for i := range s.shards {
		wg.Add(1)
		go func(sh *Shard) {
			defer wg.Done()
			sh.mu.RLock()
			for addr, p := range sh.positions {
				fn(addr, p)
			}
			sh.mu.RUnlock()
		}(&s.shards[i])
	}
	wg.Wait()
}
