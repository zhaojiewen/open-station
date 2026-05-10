package pool

import (
	"bytes"
	"testing"
)

func TestGetSmallBuffer(t *testing.T) {
	buf := GetSmallBuffer()
	if buf == nil {
		t.Error("expected buffer")
	}
	if buf.Cap() < 1024 {
		t.Errorf("expected capacity >= 1024, got %d", buf.Cap())
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty buffer, got length %d", buf.Len())
	}

	// Write and return
	buf.WriteString("test")
	PutBuffer(buf)

	// Get again should be reset
	buf2 := GetSmallBuffer()
	if buf2.Len() != 0 {
		t.Errorf("expected reset buffer, got length %d", buf2.Len())
	}
}

func TestGetMediumBuffer(t *testing.T) {
	buf := GetMediumBuffer()
	if buf == nil {
		t.Error("expected buffer")
	}
	if buf.Cap() < 16*1024 {
		t.Errorf("expected capacity >= 16KB, got %d", buf.Cap())
	}
}

func TestGetLargeBuffer(t *testing.T) {
	buf := GetLargeBuffer()
	if buf == nil {
		t.Error("expected buffer")
	}
	if buf.Cap() < 64*1024 {
		t.Errorf("expected capacity >= 64KB, got %d", buf.Cap())
	}
}

func TestGetBufferBySize(t *testing.T) {
	tests := []struct {
		size     int
		expected string
	}{
		{512, "small"},
		{1024, "small"},
		{2048, "medium"},
		{16 * 1024, "medium"},
		{32 * 1024, "large"},
		{100 * 1024, "large"},
	}

	for _, tt := range tests {
		buf := GetBuffer(tt.size)
		if buf == nil {
			t.Errorf("size %d: expected buffer", tt.size)
			continue
		}

		// Check it's from appropriate pool based on capacity
		cap := buf.Cap()
		switch tt.expected {
		case "small":
			if cap > 1024 {
				t.Errorf("size %d: expected small buffer (cap <= 1024), got cap %d", tt.size, cap)
			}
		case "medium":
			if cap > 16*1024 || cap <= 1024 {
				t.Errorf("size %d: expected medium buffer (1024 < cap <= 16KB), got cap %d", tt.size, cap)
			}
		case "large":
			if cap <= 16*1024 {
				t.Errorf("size %d: expected large buffer (cap > 16KB), got cap %d", tt.size, cap)
			}
		}
	}
}

func TestPutBufferNil(t *testing.T) {
	// Should not panic
	PutBuffer(nil)
}

func TestByteSlicePool(t *testing.T) {
	slice := GetByteSlice()
	if slice == nil {
		t.Error("expected byte slice")
	}
	if len(*slice) != 32*1024 {
		t.Errorf("expected 32KB slice, got %d", len(*slice))
	}

	PutByteSlice(slice)

	// Get again
	slice2 := GetByteSlice()
	if slice2 == nil {
		t.Error("expected byte slice")
	}
}

func TestPutByteSliceNil(t *testing.T) {
	// Should not panic
	PutByteSlice(nil)
}

func TestBufferReuse(t *testing.T) {
	// Use multiple buffers
	var buffers []*bytes.Buffer
	for i := 0; i < 10; i++ {
		buf := GetSmallBuffer()
		buf.WriteString("test data")
		buffers = append(buffers, buf)
	}

	// Put all back
	for _, buf := range buffers {
		PutBuffer(buf)
	}

	// Get again and verify reset
	for i := 0; i < 10; i++ {
		buf := GetSmallBuffer()
		if buf.Len() != 0 {
			t.Errorf("buffer %d should be reset, got length %d", i, buf.Len())
		}
		PutBuffer(buf)
	}
}