package pty

import (
	"bytes"
	"testing"
	"time"
)

// BenchmarkBufferWrite benchmarks writing to PTY buffer
func BenchmarkBufferWrite(b *testing.B) {
	buf := make([]byte, 0, BufferLimit)
	data := []byte("test output data\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if len(buf) < BufferLimit {
			buf = append(buf, data...)
		}
	}
}

// BenchmarkBufferRead benchmarks reading from PTY buffer
func BenchmarkBufferRead(b *testing.B) {
	buf := make([]byte, BufferChunk)
	for i := 0; i < BufferChunk; i++ {
		buf[i] = byte('A' + (i % 26))
	}

	dest := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		offset := (i * 1024) % (BufferChunk - 1024)
		copy(dest, buf[offset:offset+1024])
	}
}

// BenchmarkMetaFrame benchmarks creating metadata frame
func BenchmarkMetaFrame(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MetaFrame(i)
	}
}

// BenchmarkSessionCreation benchmarks creating session info
func BenchmarkSessionCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Session{
			Info: SessionInfo{
				ID:      "pty-1",
				Title:   "Terminal",
				Command: "/bin/bash",
				CWD:     "/tmp",
				Status:  StatusRunning,
				PID:     12345,
			},
			Buffer:     make([]byte, 0, BufferLimit),
			Subscribers: make(map[string]*Subscriber),
			CreatedAt:  time.Now(),
		}
	}
}

// BenchmarkSessionInfoCreation benchmarks creating session info
func BenchmarkSessionInfoCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SessionInfo{
			ID:      "pty-1",
			Title:   "Terminal",
			Command: "/bin/bash",
			CWD:     "/tmp",
			Status:  StatusRunning,
			PID:     12345,
		}
	}
}

// BenchmarkCreateInputCreation benchmarks creating input
func BenchmarkCreateInputCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CreateInput{
			Command: "/bin/bash",
			CWD:     "/tmp",
			Title:   "Terminal",
			Cols:    DefaultCols,
			Rows:    DefaultRows,
		}
	}
}

// BenchmarkSubscriberCreation benchmarks creating subscriber
func BenchmarkSubscriberCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Subscriber{
			ID:     "sub-1",
			Ready:  true,
			Cursor: 0,
		}
	}
}

// BenchmarkItoa benchmarks integer to byte conversion
func BenchmarkItoa(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		itoa(i)
	}
}

// BenchmarkMemorySession benchmarks memory usage for session creation
func BenchmarkMemorySession(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Session{
			Info: SessionInfo{
				ID:      "pty-test",
				Title:   "Test Terminal",
				Command: "/bin/bash",
				CWD:     "/tmp/test",
				Status:  StatusRunning,
			},
			Buffer:      make([]byte, 0, 1024),
			Subscribers: make(map[string]*Subscriber),
			CreatedAt:   time.Now(),
		}
	}
}

// BenchmarkBufferChunkSend benchmarks sending buffer chunks
func BenchmarkBufferChunkSend(b *testing.B) {
	buf := bytes.NewBuffer(make([]byte, BufferChunk))
	chunk := make([]byte, 64*1024) // 64KB chunk

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Read(chunk)
		buf.Write(chunk) // Re-fill for next iteration
	}
}