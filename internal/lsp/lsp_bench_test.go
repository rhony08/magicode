package lsp

import (
	"testing"
	"time"
)

// BenchmarkLanguageDetection benchmarks language detection from file path
func BenchmarkLanguageDetection(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetLanguageFromPath("/path/to/file.go")
	}
}

// BenchmarkLanguageDetectionTypescript benchmarks TS detection
func BenchmarkLanguageDetectionTypescript(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetLanguageFromPath("/path/to/file.ts")
	}
}

// BenchmarkLanguageDetectionPython benchmarks Python detection
func BenchmarkLanguageDetectionPython(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetLanguageFromPath("/path/to/file.py")
	}
}

// BenchmarkServerID benchmarks using server ID
func BenchmarkServerID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ServerGo
	}
}

// BenchmarkServerIDTypescript benchmarks using TS server ID
func BenchmarkServerIDTypescript(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ServerTypeScript
	}
}

// BenchmarkFileStateCreation benchmarks creating file state
func BenchmarkFileStateCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FileState{
			URI:        "file:///path/to/file.go",
			Version:    1,
			LastAccess: time.Now(),
		}
	}
}

// BenchmarkDiagnosticCreation benchmarks creating diagnostic
func BenchmarkDiagnosticCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Diagnostic{
			Range: Range{
				Start: Position{Line: 1, Character: 0},
				End:   Position{Line: 1, Character: 10},
			},
			Message:  "Test diagnostic",
			Severity: SeverityError,
			Source:   "test",
		}
	}
}

// BenchmarkPositionCreation benchmarks creating position
func BenchmarkPositionCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Position{Line: 1, Character: 0}
	}
}

// BenchmarkRangeCreation benchmarks creating range
func BenchmarkRangeCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Range{
			Start: Position{Line: 1, Character: 0},
			End:   Position{Line: 2, Character: 10},
		}
	}
}

// BenchmarkMemoryDiagnostic benchmarks memory usage for diagnostics
func BenchmarkMemoryDiagnostic(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Diagnostic{
			Range: Range{
				Start: Position{Line: 1, Character: 0},
				End:   Position{Line: 1, Character: 10},
			},
			Message:  "Test diagnostic message",
			Severity: SeverityError,
			Source:   "benchmark",
		}
	}
}