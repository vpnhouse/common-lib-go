package xpipe

import (
	"bytes"
	"errors"
	"os"
	"sync"
	"testing"

	"math/rand"

	"github.com/stretchr/testify/assert"
)

const (
	testRounds  = 10000
	messageSize = 1024 * 1024
	initialSeed = 12345

	writeMaxSize = 65536
	readMaxSize  = 65536
)

var (
	randInstance = rand.New(rand.NewSource(initialSeed))
)

func once(t *testing.T) {
	originalBuffer := make([]byte, messageSize)

	n, err := randInstance.Read(originalBuffer)
	assert.Nil(t, err)
	assert.Equal(t, messageSize, n)

	pipe, err := New()
	assert.Nil(t, err)

	wg := sync.WaitGroup{}
	wg.Add(2)
	n, err = pipe.Write(make([]byte, 0))
	assert.Nil(t, err)
	assert.Zero(t, n)
	go func() {
		defer pipe.Close()
		defer wg.Done()
		src := bytes.NewBuffer(originalBuffer)
		for {
			buf := make([]byte, rand.Intn(writeMaxSize-1)+1)
			nR, err := src.Read(buf)
			if nR == 0 {
				break
			}

			assert.Nil(t, err)

			nW, err := pipe.Write(buf[:nR])
			assert.Nil(t, err)

			assert.Equal(t, nR, nW)
		}
	}()

	var copiedBuffer []byte = make([]byte, 0)
	n, err = pipe.Read(make([]byte, 0))
	assert.Nil(t, err)
	assert.Zero(t, n)

	go func() {
		defer pipe.Close()
		defer wg.Done()
		for {
			chunkSize := rand.Intn(readMaxSize-1) + 1
			buf := make([]byte, chunkSize)
			n, err := pipe.Read(buf)
			if errors.Is(err, os.ErrClosed) {
				break
			}
			assert.Nil(t, err)

			assert.Greater(t, n, 0)
			assert.LessOrEqual(t, n, chunkSize)
			copiedBuffer = append(copiedBuffer, buf[:n]...)
		}
	}()
	wg.Wait()

	broken := false
	assert.Equal(t, len(originalBuffer), len(copiedBuffer))
	for idx, v := range originalBuffer {
		if v != copiedBuffer[idx] {
			broken = true
		}
	}
	// if broken {appe
	// 	fmt.Println("========================== ORIGINAL ==========================")
	// 	fmt.Println(originalBuffer)
	// 	fmt.Println("========================== COPIED ==========================")
	// 	fmt.Println(copiedBuffer)
	// }
	assert.False(t, broken)

}

func TestGeneric(t *testing.T) {
	for idx := 0; idx < testRounds; idx++ {
		once(t)
	}
}
