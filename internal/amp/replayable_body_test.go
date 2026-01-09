package amp

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
)

func TestNewReplayableBody(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxSize  int64
		wantErr  error
		wantSize int64
	}{
		{
			name:     "normal body",
			input:    "hello world",
			maxSize:  1024,
			wantErr:  nil,
			wantSize: 11,
		},
		{
			name:     "empty body",
			input:    "",
			maxSize:  1024,
			wantErr:  nil,
			wantSize: 0,
		},
		{
			name:     "body at limit",
			input:    strings.Repeat("x", 100),
			maxSize:  100,
			wantErr:  nil,
			wantSize: 100,
		},
		{
			name:     "body exceeds limit",
			input:    strings.Repeat("x", 101),
			maxSize:  100,
			wantErr:  ErrBodyTooLarge,
			wantSize: 0,
		},
		{
			name:     "default max size",
			input:    "test",
			maxSize:  0, // should use default
			wantErr:  nil,
			wantSize: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := io.NopCloser(strings.NewReader(tt.input))
			rb, err := NewReplayableBody(r, tt.maxSize)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if rb.Size() != tt.wantSize {
				t.Errorf("expected size %d, got %d", tt.wantSize, rb.Size())
			}
		})
	}
}

func TestReplayableBody_NewReader(t *testing.T) {
	input := "hello replayable body"
	r := io.NopCloser(strings.NewReader(input))
	rb, err := NewReplayableBody(r, 1024)
	if err != nil {
		t.Fatalf("failed to create ReplayableBody: %v", err)
	}

	// Read multiple times
	for i := 0; i < 3; i++ {
		reader := rb.NewReader()
		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("read %d failed: %v", i, err)
		}
		if string(data) != input {
			t.Errorf("read %d: expected %q, got %q", i, input, string(data))
		}
	}
}

func TestReplayableBody_ConcurrentRead(t *testing.T) {
	input := strings.Repeat("concurrent test ", 100)
	r := io.NopCloser(strings.NewReader(input))
	rb, err := NewReplayableBody(r, int64(len(input)*2))
	if err != nil {
		t.Fatalf("failed to create ReplayableBody: %v", err)
	}

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reader := rb.NewReader()
			data, err := io.ReadAll(reader)
			if err != nil {
				errors <- err
				return
			}
			if string(data) != input {
				errors <- bytes.ErrTooLarge
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent read error: %v", err)
	}
}

func TestReplayableBodyWithTruncation(t *testing.T) {
	input := strings.Repeat("x", 200)
	maxSize := int64(100)

	r := io.NopCloser(strings.NewReader(input))
	rb, err := NewReplayableBodyWithTruncation(r, maxSize)
	if err != nil {
		t.Fatalf("failed to create ReplayableBody: %v", err)
	}

	if !rb.IsTruncated() {
		t.Error("expected body to be truncated")
	}

	if rb.Size() != maxSize {
		t.Errorf("expected size %d, got %d", maxSize, rb.Size())
	}

	data := rb.Bytes()
	if int64(len(data)) != maxSize {
		t.Errorf("expected data length %d, got %d", maxSize, len(data))
	}
}

func TestReplayableBody_Bytes(t *testing.T) {
	input := "test bytes"
	r := io.NopCloser(strings.NewReader(input))
	rb, err := NewReplayableBody(r, 1024)
	if err != nil {
		t.Fatalf("failed to create ReplayableBody: %v", err)
	}

	// Bytes returns reference
	data := rb.Bytes()
	if string(data) != input {
		t.Errorf("expected %q, got %q", input, string(data))
	}

	// BytesCopy returns copy
	dataCopy := rb.BytesCopy()
	if string(dataCopy) != input {
		t.Errorf("expected %q, got %q", input, string(dataCopy))
	}
}

func TestReplayableBody_NilReader(t *testing.T) {
	rb, err := NewReplayableBody(nil, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !rb.IsEmpty() {
		t.Error("expected empty body")
	}

	reader := rb.NewReader()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty data, got %d bytes", len(data))
	}
}

func TestReplayableBody_Clear(t *testing.T) {
	input := "test clear"
	r := io.NopCloser(strings.NewReader(input))
	rb, err := NewReplayableBody(r, 1024)
	if err != nil {
		t.Fatalf("failed to create ReplayableBody: %v", err)
	}

	if rb.IsEmpty() {
		t.Error("expected non-empty body before clear")
	}

	rb.Clear()

	if !rb.IsEmpty() {
		t.Error("expected empty body after clear")
	}
	if rb.Size() != 0 {
		t.Error("expected size 0 after clear")
	}
}

func TestReplayableBody_String(t *testing.T) {
	input := "test string method"
	r := io.NopCloser(strings.NewReader(input))
	rb, err := NewReplayableBody(r, 1024)
	if err != nil {
		t.Fatalf("failed to create ReplayableBody: %v", err)
	}

	if rb.String() != input {
		t.Errorf("expected %q, got %q", input, rb.String())
	}
}

func BenchmarkReplayableBody_NewReader(b *testing.B) {
	input := strings.Repeat("benchmark data ", 1000)
	r := io.NopCloser(strings.NewReader(input))
	rb, _ := NewReplayableBody(r, int64(len(input)*2))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := rb.NewReader()
		_, _ = io.ReadAll(reader)
	}
}

func BenchmarkReplayableBody_Create(b *testing.B) {
	input := strings.Repeat("benchmark data ", 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := io.NopCloser(strings.NewReader(input))
		_, _ = NewReplayableBody(r, int64(len(input)*2))
	}
}
