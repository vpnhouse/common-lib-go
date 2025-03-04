package xcache

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestXCacheSetGet(t *testing.T) {
	c, err := New(1, nil)
	assert.NoError(t, err)

	defer c.Reset()

	v, err := c.Get(nil)
	assert.ErrorIs(t, err, ErrNoData)
	assert.Equalf(t, 0, len(v), "unexpected non-empty value obtained from cache: %q", v)

	c.Set([]byte("key"), []byte("value"))
	v, err = c.Get([]byte("key"))
	assert.NoError(t, err)
	assert.Equalf(t, string(v), "value", "unexpected value obtained; got %q; want %q for key: %q", v, "value", []byte("key"))

	v, err = c.Get(nil)
	assert.ErrorIs(t, err, ErrNoData)
	assert.Equalf(t, 0, len(v), "unexpected non-empty value obtained from cache: %q by key: nil", v)

	v, err = c.Get([]byte("aaa"))
	assert.ErrorIs(t, err, ErrNoData)
	assert.Equalf(t, 0, len(v), "unexpected non-empty value obtained from cache: %q by key: %q", v, []byte("aaa"))

	err = c.Set([]byte("aaa"), []byte("bbb"))
	assert.NoError(t, err)

	v, err = c.Get([]byte("aaa"))
	assert.NoError(t, err)
	assert.Equalf(t, string(v), "bbb", "unexpected value obtained; got %q; want %q for key: %q", v, "value", []byte("key"))

	c.Reset()

	v, err = c.Get([]byte("aaa"))
	assert.ErrorIs(t, err, ErrNoData)
	assert.Equalf(t, 0, len(v), "unexpected non-empty value obtained after reset from cache: %q by key: %q", v, []byte("aaa"))

	// Test empty value
	k := []byte("empty")
	err = c.Set(k, nil)
	assert.NoError(t, err)

	v, err = c.Get(k)
	assert.NoError(t, err)
	assert.Equalf(t, 0, len(v), "unexpected non-empty value obtained from cache: %q by key: %q", v, k)

	v, err = c.Get([]byte("foobar"))
	assert.ErrorIs(t, err, ErrNoData)
	assert.Equalf(t, 0, len(v), "unexpected non-empty value obtained from cache: %q by key: %q", v, k)
}

func TestXCacheDel(t *testing.T) {
	c, err := New(1, nil)
	assert.NoError(t, err)

	defer c.Reset()

	for i := 0; i < 100; i++ {
		k := []byte(fmt.Sprintf("key %d", i))
		v := []byte(fmt.Sprintf("value %d", i))
		err := c.Set(k, v)
		assert.NoError(t, err)

		vv, err := c.Get(k)
		assert.NoError(t, err)
		assert.Equalf(t, string(vv), string(v), "unexpected value for key %q; got %q; want %q", k, vv, v)

		c.Del(k)
		vv, err = c.Get(k)
		assert.ErrorIs(t, err, ErrNoData)
		assert.Equalf(t, 0, len(vv), "unexpected non-empty value got for key %q: %q", k, vv)
	}
}

func TestXCacheUpdate(t *testing.T) {
	c, err := New(1, nil)
	assert.NoError(t, err)

	k := []byte("aaa")
	v := []byte("bbb")
	err = c.Set(k, v)
	assert.NoError(t, err)

	err = c.Update(k, func(v []byte) ([]byte, bool, error) {
		assert.Equal(t, v, []byte("bbb"))
		v[0] = 'a'
		v[1] = 'c'
		v[2] = 'd'
		return v, false, nil
	})
	assert.NoError(t, err)

	vv, err := c.Get(k, true)
	assert.NoError(t, err)
	assert.Equal(t, []byte("acd"), vv)
}

func BenchmarkCacheSet(b *testing.B) {
	const items = 1 << 16
	c, _ := New(12*items, nil)
	defer c.Reset()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		k := []byte("\x00\x00")
		v := []byte("\x00\x00")
		for pb.Next() {
			for i := 0; i < items; i++ {
				k[0]++
				v[0]++
				if k[0] == 0 {
					k[1]++
					v[1]++
				}
				err := c.Set(k, v)
				assert.NoError(b, err)

			}
		}
	})
}

func TestEvict(t *testing.T) {
	testItems := map[string]string{
		"aaa": "bbb",
		"ccc": "ddd",
		"fff": "eee",
	}

	evictedItems := make(map[string]string, len(testItems))

	done := make(chan struct{})

	onEvict := func(items *Items) {
		for i := range items.Keys {
			evictedItems[string(items.Keys[i])] = string(items.Values[i])
		}
		close(done)
	}

	c, _ := New(1, onEvict)
	for k, v := range testItems {
		err := c.Set([]byte(k), []byte(v))
		assert.NoErrorf(t, err, "failed to add to cache: %s = %s", k, v)
	}

	c.Reset()
	<-done

	assert.Equal(t, testItems, evictedItems)
}

func BenchmarkCacheGet(b *testing.B) {
	const items = 1 << 16
	c, _ := New(12*items, nil)
	defer c.Reset()
	k := []byte("\x00\x00")
	v := []byte("\x00\x00")
	for range items {
		k[0]++
		v[0]++
		if k[0] == 0 {
			k[1]++
			v[1]++
		}
		err := c.Set(k, v)
		assert.NoError(b, err)
		if err != nil {
			b.Fatal()
		}
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		k := []byte("\x00\x00")
		v := []byte("\x00\x00")
		for pb.Next() {
			for range items {
				k[0]++
				v[0]++
				if k[0] == 0 {
					k[1]++
					v[1]++
				}
				vv, err := c.Get(k, true)
				assert.NoError(b, err)
				assert.Truef(b, bytes.Equal(vv, v), "BUG: invalid value obtained; got %q; want %q", vv, v)
				if !bytes.Equal(vv, v) {
					b.Fatal()
				}
			}
		}
	})
}
