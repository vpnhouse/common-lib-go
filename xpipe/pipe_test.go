package xpipe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"math/rand"

	"github.com/stretchr/testify/assert"
)

const (
	testRounds  = 1000
	messageSize = 1024 * 1024
	initialSeed = 12345

	writeMaxSize = 65536
	readMaxSize  = 65536
	testTimeout  = 20 * time.Second
)

var (
	randInstance = rand.New(rand.NewSource(initialSeed))
	ctx, _       = context.WithTimeout(context.Background(), testTimeout)
)

type lenGetnType func() int

func once(t *testing.T, rlen, wlen lenGetnType) {
	originalBuffer := make([]byte, messageSize)

	n, err := randInstance.Read(originalBuffer)
	assert.Nil(t, err)
	assert.Equal(t, messageSize, n)

	pipe, err := New(ctx)
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
			buf := make([]byte, wlen())
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
			chunkSize := rlen()
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
	assert.False(t, broken)

}

func TestPerByte(t *testing.T) {
	fmt.Println("Byte-transfer test")

	once(t,
		func() int { return 1 },
		func() int { return 1 },
	)
}

func TestDeadlock(t *testing.T) {
	fmt.Println("Deaclock test")
	wg := sync.WaitGroup{}

	pipe, err := New(ctx)
	assert.Nil(t, err)
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(time.Millisecond * 100)
		buf := make([]byte, 16)

		n, err := pipe.Read(buf)
		assert.Equal(t, 0, n)
		assert.ErrorIs(t, err, os.ErrClosed)
	}()
	pipe.Close()
	err = pipe.Close()
	assert.ErrorIs(t, err, os.ErrClosed)
	wg.Wait()

	pipe, err = New(ctx)
	assert.Nil(t, err)
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(time.Millisecond * 100)
		buf := make([]byte, 1024*1024)
		n, err := pipe.Write(buf)
		assert.Equal(t, 0, n)
		assert.ErrorIs(t, err, os.ErrClosed)
	}()
	pipe.Close()
	err = pipe.Close()
	assert.ErrorIs(t, err, os.ErrClosed)
	wg.Wait()
}

func TestGeneric(t *testing.T) {
	fmt.Println("Random size read/write test in", testRounds, "rounds")
	for idx := 0; idx < testRounds; idx++ {
		once(t,
			func() int { return rand.Intn(readMaxSize-1) + 1 },
			func() int { return rand.Intn(writeMaxSize-1) + 1 },
		)
	}
}

func TestPerformance(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	p, err := New(ctx)
	assert.Nil(t, err)

	src := make([]byte, 1001)
	dst := make([]byte, 1000)
	received := 0
	go func() {
		for {
			n, err := p.Read(dst)
			received += n
			if err != nil {
				return
			}
		}
	}()

	for {
		_, err := p.Write(src)
		if err != nil {
			break
		}
	}

	assert.Greater(t, received, 10000000)

	fmt.Println("Performance test:", received/1000000, "MBps")
}
