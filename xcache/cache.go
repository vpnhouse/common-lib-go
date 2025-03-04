// Inspired by  https://victoriametrics.com/
package xcache

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	xxhash "github.com/cespare/xxhash/v2"
)

const (
	bucketsCount          = 512
	chunkSize             = 64 * 1024
	bucketSizeBits        = 40
	genSizeBits           = 64 - bucketSizeBits
	maxGen                = 1<<genSizeBits - 1
	maxBucketSize  uint64 = 1 << bucketSizeBits
	maxKeyLen             = 1 << 16
	maxValLen             = 1 << 16
)

var (
	ErrNoData        = errors.New("no data")
	ErrCorruptedData = errors.New("corrupted data")
)

var (
	errKeyLenTooBig              = errors.New("key len too big > 65536")
	errValLenTooBig              = errors.New("val len too big > 65536")
	errKeyValLenExceedsChunkSize = errors.New("key + val len exceeds chunk size")
)

type Items struct {
	Keys   [][]byte
	Values [][]byte
}

func newItems(size int) *Items {
	if size <= 0 {
		return &Items{}
	}
	return &Items{
		Keys:   make([][]byte, 0, size),
		Values: make([][]byte, 0, size),
	}
}

func (i *Items) Add(key []byte, val []byte) {
	i.Keys = append(i.Keys, key)
	i.Values = append(i.Values, val)
}

func (i *Items) Count() int {
	return len(i.Keys)
}

type (
	OnEvict func(items *Items)
	Mutator func(v []byte) ([]byte, bool, error)
)

// Thread-safe inmemory cache optimized for big number
// of entries.
type Cache struct {
	buckets [bucketsCount]bucket
	onEvict OnEvict
}

// New Cache
// If maxBytes is less than 32MB, then the minimum cache capacity is 32MB.
func New(maxBytes int, onEvict OnEvict) (*Cache, error) {
	if maxBytes <= 0 {
		return nil, fmt.Errorf("maxBytes must be greater than 0; got %d", maxBytes)
	}
	c := Cache{
		onEvict: onEvict,
	}
	maxBucketBytes := uint64((maxBytes + bucketsCount - 1) / bucketsCount)
	for i := range c.buckets[:] {
		err := c.buckets[i].Init(maxBucketBytes)
		if err != nil {
			return nil, err
		}
	}
	return &c, nil
}

// Stores (k, v) in the cache with given mutator if any
func (c *Cache) Update(k []byte, mutator Mutator) error {
	h := xxhash.Sum64(k)
	idx := h % bucketsCount
	return c.buckets[idx].Update(k, h, mutator, c.onEvict)
}

// Stores (k, v) in the cache
func (c *Cache) Set(k, v []byte) error {
	h := xxhash.Sum64(k)
	idx := h % bucketsCount
	return c.buckets[idx].Set(k, v, h, c.onEvict)
}

func (c *Cache) Get(k []byte, noCopy ...bool) ([]byte, error) {
	h := xxhash.Sum64(k)
	idx := h % bucketsCount
	noCopyVal := len(noCopy) > 0 && noCopy[0]
	return c.buckets[idx].Get(k, h, noCopyVal)
}

// Del deletes value for the given k from the cache.
func (c *Cache) Del(k []byte) {
	h := xxhash.Sum64(k)
	idx := h % bucketsCount
	c.buckets[idx].Del(h)
}

// Reset removes all the items from the cache.
func (c *Cache) Reset() {
	var evicted *Items
	if c.onEvict != nil {
		evicted = newItems(0)
	}
	for i := range c.buckets[:] {
		c.buckets[i].Reset(evicted)
	}

	if c.onEvict != nil && evicted.Count() > 0 {
		go c.onEvict(evicted)
	}
}

type bucket struct {
	l sync.Mutex

	// chunks is a ring buffer with encoded (k, v) pairs.
	// It consists of 64KB chunks.
	chunks [][]byte

	// m maps hash(k) to idx of (k, v) pair in chunks.
	m map[uint64]uint64

	// idx points to chunks for writing the next (k, v) pair.
	idx uint64

	// gen is the generation of chunks.
	gen uint64
}

func (b *bucket) Init(maxBytes uint64) error {
	if maxBytes == 0 {
		return fmt.Errorf("maxBytes cannot be zero")
	}
	if maxBytes >= maxBucketSize {
		return fmt.Errorf("too big maxBytes=%d; should be smaller than %d", maxBytes, maxBucketSize)
	}
	maxChunks := (maxBytes + chunkSize - 1) / chunkSize
	b.chunks = make([][]byte, maxChunks)
	b.m = make(map[uint64]uint64)
	b.Reset(nil)
	return nil
}

func (b *bucket) Reset(out *Items) {
	b.l.Lock()
	defer b.l.Unlock()

	if out != nil {
		for _, v := range b.m {
			key, val, err := b.readItem(v)
			if err != nil {
				continue
			}
			out.Add(key, val)
		}
	}

	for i := range b.chunks {
		putChunk(b.chunks[i])
		b.chunks[i] = nil
	}
	b.m = make(map[uint64]uint64)
	b.idx = 0
	b.gen = 1
}

func (b *bucket) cleanLocked(onEvict OnEvict) {
	bGen := b.gen & maxGen
	// Re-create b.m with valid items, which weren't expired yet instead of deleting expired items from b.m.
	// This should reduce memory fragmentation and the number Go objects behind b.m.
	// See https://github.com/VictoriaMetrics/VictoriaMetrics/issues/5379
	var bm map[uint64]uint64
	var bme map[uint64]struct{} // deleted items
	for k, v := range b.m {
		gen := v >> bucketSizeBits
		idx := v & (maxBucketSize - 1)
		if (gen+1 == bGen || gen == maxGen && bGen == 1) && idx >= b.idx || gen == bGen && idx < b.idx {
			if bm == nil {
				bm = make(map[uint64]uint64, len(b.m))
			}
			bm[k] = v
		} else if onEvict != nil {
			if bme == nil {
				bme = make(map[uint64]struct{}, len(b.m))
			}
			bme[v] = struct{}{}
		}
	}
	if onEvict != nil && bme != nil {
		evicted := newItems(len(bme))
		for v := range bme {
			key, val, err := b.readItem(v)
			if err != nil {
				continue
			}
			evicted.Add(key, val)
		}
		go onEvict(evicted)
	}
	if bm != nil {
		b.m = bm
	}
}

func (b *bucket) Set(k, v []byte, h uint64, onEvict OnEvict) error {
	b.l.Lock()
	defer b.l.Unlock()
	return b.setLocked(k, v, h, onEvict)
}

func (b *bucket) GetItems() *Items {
	b.l.Lock()
	defer b.l.Unlock()
	return b.getItemsLocked(nil)
}

func (b *bucket) getItemsLocked(out *Items) *Items {
	if len(b.m) == 0 {
		return out
	}
	if out == nil {
		out = newItems(len(b.m))
	}
	for _, v := range b.m {
		key, val, err := b.readItem(v)
		if err != nil {
			continue
		}
		out.Add(key, val)
	}
	return out
}

func (b *bucket) setLocked(k, v []byte, h uint64, onEvict OnEvict) error {
	// Too big key or value - its length cannot be encoded
	// with 2 bytes (see below). Skip the entry.
	if len(k) >= maxKeyLen {
		return errKeyLenTooBig
	}
	if len(v) >= maxKeyLen {
		return errValLenTooBig
	}
	var kvLenBuf [4]byte
	kvLenBuf[0] = byte(uint16(len(k)) >> 8)
	kvLenBuf[1] = byte(len(k))
	kvLenBuf[2] = byte(uint16(len(v)) >> 8)
	kvLenBuf[3] = byte(len(v))
	kvLen := uint64(len(kvLenBuf) + len(k) + len(v))
	if kvLen >= chunkSize {
		// Do not store too big keys and values, since they do not
		// fit a chunk.
		return errKeyValLenExceedsChunkSize
	}

	chunks := b.chunks
	needClean := false

	idx := b.idx
	idxNew := idx + kvLen
	chunkIdx := idx / chunkSize
	chunkIdxNew := idxNew / chunkSize
	if chunkIdxNew > chunkIdx {
		if chunkIdxNew >= uint64(len(chunks)) {
			idx = 0
			idxNew = kvLen
			chunkIdx = 0
			b.gen++
			if b.gen&maxGen == 0 {
				b.gen++
			}
			needClean = true
		} else {
			idx = chunkIdxNew * chunkSize
			idxNew = idx + kvLen
			chunkIdx = chunkIdxNew
		}
		chunks[chunkIdx] = chunks[chunkIdx][:0]
	}
	chunk := chunks[chunkIdx]
	if chunk == nil {
		chunk = getChunk()
		chunk = chunk[:0]
	}
	chunk = append(chunk, kvLenBuf[:]...)
	chunk = append(chunk, k...)
	chunk = append(chunk, v...)
	chunks[chunkIdx] = chunk
	b.m[h] = idx | (b.gen << bucketSizeBits)
	b.idx = idxNew
	if needClean {
		b.cleanLocked(onEvict)
	}
	return nil
}

func (b *bucket) Get(k []byte, h uint64, noCopy bool) ([]byte, error) {
	b.l.Lock()
	defer b.l.Unlock()
	return b.getLocked(k, h, noCopy)
}

func (b *bucket) readItem(v uint64) ([]byte, []byte, error) {
	gen := v >> bucketSizeBits
	idx := v & (maxBucketSize - 1)
	bGen := b.gen & maxGen

	if !(gen == bGen && idx < b.idx || gen+1 == bGen && idx >= b.idx || gen == maxGen && bGen == 1 && idx >= b.idx) {
		return nil, nil, ErrNoData
	}

	chunkIdx := idx / chunkSize
	if chunkIdx >= uint64(len(b.chunks)) {
		// Corrupted data
		return nil, nil, ErrCorruptedData
	}
	chunk := b.chunks[chunkIdx]
	idx %= chunkSize
	if idx+4 >= chunkSize {
		// Corrupted data
		return nil, nil, ErrCorruptedData
	}
	kvLenBuf := chunk[idx : idx+4]
	keyLen := (uint64(kvLenBuf[0]) << 8) | uint64(kvLenBuf[1])
	valLen := (uint64(kvLenBuf[2]) << 8) | uint64(kvLenBuf[3])
	idx += 4
	if idx+keyLen+valLen >= chunkSize {
		// Corrupted data
		return nil, nil, ErrCorruptedData
	}
	k := make([]byte, keyLen)
	copy(k, chunk[idx:idx+keyLen])
	idx += keyLen

	val := make([]byte, valLen)
	copy(val, chunk[idx:idx+valLen])

	return k, val, nil
}

func (b *bucket) getLocked(k []byte, h uint64, noCopy bool) ([]byte, error) {
	v := b.m[h]
	if v == 0 {
		return nil, ErrNoData
	}
	bGen := b.gen & maxGen

	gen := v >> bucketSizeBits
	idx := v & (maxBucketSize - 1)

	if !(gen == bGen && idx < b.idx || gen+1 == bGen && idx >= b.idx || gen == maxGen && bGen == 1 && idx >= b.idx) {
		return nil, ErrNoData
	}

	chunkIdx := idx / chunkSize
	if chunkIdx >= uint64(len(b.chunks)) {
		// Corrupted data
		return nil, ErrCorruptedData
	}
	chunk := b.chunks[chunkIdx]
	idx %= chunkSize
	if idx+4 >= chunkSize {
		// Corrupted data
		return nil, ErrCorruptedData
	}
	kvLenBuf := chunk[idx : idx+4]
	keyLen := (uint64(kvLenBuf[0]) << 8) | uint64(kvLenBuf[1])
	valLen := (uint64(kvLenBuf[2]) << 8) | uint64(kvLenBuf[3])
	idx += 4
	if idx+keyLen+valLen >= chunkSize {
		// Corrupted data
		return nil, ErrCorruptedData
	}
	if bytes.Equal(k, chunk[idx:idx+keyLen]) {
		idx += keyLen
		if noCopy {
			return chunk[idx : idx+valLen], nil
		}
		val := make([]byte, 0, valLen)
		return append(val, chunk[idx:idx+valLen]...), nil
	}
	return nil, ErrNoData
}

func (b *bucket) Update(k []byte, h uint64, mutator Mutator, onEvict OnEvict) error {
	b.l.Lock()
	defer b.l.Unlock()

	v, err := b.getLocked(k, h, true)
	if err == nil || errors.Is(err, ErrNoData) {
		v, update, err := mutator(v)
		if err != nil {
			return err
		}
		if update || errors.Is(err, ErrNoData) {
			return b.setLocked(k, v, h, onEvict)
		}
	}
	return err
}

func (b *bucket) Del(h uint64) {
	b.l.Lock()
	defer b.l.Unlock()
	delete(b.m, h)
}
