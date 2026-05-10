package pool

import (
	"bytes"
	"sync"
)

// BufferPool provides pooled byte buffers for high-throughput operations.
// This reduces GC pressure by reusing buffers instead of allocating new ones.
var (
	// Small buffers for headers, keys, short responses (1KB)
	smallBufferPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 0, 1024))
		},
	}

	// Medium buffers for request bodies, JSON encoding (16KB)
	mediumBufferPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 0, 16*1024))
		},
	}

	// Large buffers for streaming chunks (64KB)
	largeBufferPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 0, 64*1024))
		},
	}
)

// GetSmallBuffer returns a small buffer (1KB capacity) from the pool.
// Call PutBuffer after use.
func GetSmallBuffer() *bytes.Buffer {
	buf := smallBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// GetMediumBuffer returns a medium buffer (16KB capacity) from the pool.
// Call PutBuffer after use.
func GetMediumBuffer() *bytes.Buffer {
	buf := mediumBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// GetLargeBuffer returns a large buffer (64KB capacity) from the pool.
// Call PutBuffer after use.
func GetLargeBuffer() *bytes.Buffer {
	buf := largeBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// GetBuffer returns a buffer with approximately the requested capacity.
// It chooses the smallest pool that can accommodate the size.
func GetBuffer(size int) *bytes.Buffer {
	switch {
	case size <= 1024:
		return GetSmallBuffer()
	case size <= 16*1024:
		return GetMediumBuffer()
	default:
		return GetLargeBuffer()
	}
}

// PutBuffer returns a buffer to the appropriate pool.
// Do not use the buffer after putting it back.
func PutBuffer(buf *bytes.Buffer) {
	if buf == nil {
		return
	}

	cap := buf.Cap()
	buf.Reset()

	switch {
	case cap <= 1024:
		smallBufferPool.Put(buf)
	case cap <= 16*1024:
		mediumBufferPool.Put(buf)
	default:
		largeBufferPool.Put(buf)
	}
}

// ByteSlicePool provides pooled byte slices for reading stream data.
var byteSlicePool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 32*1024) // 32KB read buffer
		return &b
	},
}

// GetByteSlice returns a pooled byte slice (32KB) for reading.
// Call PutByteSlice after use.
func GetByteSlice() *[]byte {
	return byteSlicePool.Get().(*[]byte)
}

// PutByteSlice returns a byte slice to the pool.
func PutByteSlice(b *[]byte) {
	if b != nil {
		byteSlicePool.Put(b)
	}
}