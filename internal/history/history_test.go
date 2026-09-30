package history

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type historyReader struct {
	*strings.Reader
	bytesRead int
}

func (r *historyReader) ReadAt(p []byte, offset int64) (int, error) {
	n, err := r.Reader.ReadAt(p, offset)
	r.bytesRead += n
	return n, err
}

func TestRecentHistoryDoesNotReadOldRecords(t *testing.T) {
	line := "{\"cmd\":\"hello\"}\n"
	r := &historyReader{Reader: strings.NewReader(strings.Repeat(line, 100000))}
	entries, err := loadRecent(r, r.Size(), 1000)
	if err != nil || len(entries) != 1000 {
		t.Fatalf("load: %d, %v", len(entries), err)
	}
	if r.bytesRead > 64*1024 {
		t.Fatalf("read %d bytes for a tail smaller than one chunk", r.bytesRead)
	}
}

func TestLoadRecentHistory(t *testing.T) {
	for _, trailing := range []string{"", "\n"} {
		path := filepath.Join(t.TempDir(), "history.jsonl")
		spanningRecord, _ := json.Marshal(Entry{Command: strings.Repeat("x", 100000)})
		content := "{\"cmd\":\"first\"}\n\ninvalid json\n" + string(spanningRecord) + "\n{\"cmd\":\"third\"}\n{broken\n{\"cmd\":\"last\"}" + trailing
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		all, err := Load(path, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(all) != 4 || all[0].Command != "last" || all[3].Command != "first" {
			t.Fatalf("unexpected history: %d entries", len(all))
		}
		for _, limit := range []int{-1, 1, 2, 3, 4, 10} {
			got, err := Load(path, limit)
			if err != nil {
				t.Fatal(err)
			}
			want := all
			if limit > 0 {
				want = all[:min(limit, len(all))]
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("limit %d: bounded and unlimited loads differ", limit)
			}
		}
	}
}

func TestLoadEmptyAndMissingHistory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.jsonl")
	got, err := Load(path, 10)
	if err != nil || len(got) != 0 {
		t.Fatalf("missing: %v, %v", got, err)
	}
	for _, content := range []string{"", "\n\n", "not json\n"} {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path, 10)
		if err != nil || len(got) != 0 {
			t.Fatalf("empty: %v, %v", got, err)
		}
	}
}

func BenchmarkLoadRecent(b *testing.B) {
	for _, count := range []int{1000, 100000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			path := filepath.Join(b.TempDir(), "history.jsonl")
			line := "{\"ts\":\"2026-01-01T00:00:00Z\",\"cmd\":\"echo hello\",\"file\":\"greeting.md\",\"header\":\"Greeting\",\"scope\":{\"value\":\"hello\"}}\n"
			if err := os.WriteFile(path, []byte(strings.Repeat(line, count)), 0600); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				entries, err := Load(path, 1000)
				if err != nil || len(entries) != 1000 {
					b.Fatalf("load: %d, %v", len(entries), err)
				}
			}
		})
	}
}

func TestLoadRecentLineSizeLimit(t *testing.T) {
	for _, ending := range []string{"", "\n", "\r\n"} {
		for _, size := range []int{1024*1024 - 1, 1024 * 1024, 1024*1024 + 1} {
			t.Run(fmt.Sprintf("size=%d/ending=%q", size, ending), func(t *testing.T) {
				command := strings.Repeat("x", size-len(`{"cmd":""}`))
				content := `{"cmd":"` + command + `"}` + ending
				path := filepath.Join(t.TempDir(), "history.jsonl")
				if err := os.WriteFile(path, []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
				for _, limit := range []int{0, 1} {
					got, err := Load(path, limit)
					oversized := size >= 1024*1024 || size == 1024*1024-1 && ending == "\r\n"
					if oversized {
						if !errors.Is(err, bufio.ErrTooLong) {
							t.Fatalf("limit %d: expected line limit error, got %v", limit, err)
						}
					} else if err != nil || len(got) != 1 || got[0].Command != command {
						t.Fatalf("limit %d: expected one complete record, got %d, %v", limit, len(got), err)
					}
				}
			})
		}
	}
}

func TestLoadRecentStopsBeforeOldOversizedRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.jsonl")
	content := strings.Repeat("x", 1024*1024+1) + "\n{\"cmd\":\"recent\"}\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path, 1)
	if err != nil || len(got) != 1 || got[0].Command != "recent" {
		t.Fatalf("expected recent entry without reading old record: %v, %v", got, err)
	}
	if _, err := Load(path, 0); !errors.Is(err, bufio.ErrTooLong) {
		t.Fatalf("unlimited load must still report oversized old record: %v", err)
	}
}

func TestLoadRecentSkipsMalformedRecordsAcrossChunks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.jsonl")
	content := "{\"cmd\":\"first\"}\r\n{\"cmd\":\"second\"}\r\n" + strings.Repeat("invalid\r\n\r\n", 10000) + "{\"cmd\":\"last\"}\r\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path, 2)
	want := []Entry{{Command: "last"}, {Command: "second"}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("expected newest valid entries: %v, %v", got, err)
	}
}

func TestRecentHistorySnapshot(t *testing.T) {
	first := "{\"cmd\":\"first\"}\n"
	r := strings.NewReader(first + "{\"cmd\":\"later\"}\n")
	got, err := loadRecent(r, int64(len(first)), 10)
	if err != nil || !reflect.DeepEqual(got, []Entry{{Command: "first"}}) {
		t.Fatalf("snapshot included later append: %v, %v", got, err)
	}
	if _, err := loadRecent(r, r.Size()+1, 10); !errors.Is(err, io.EOF) {
		t.Fatalf("expected truncated read error: %v", err)
	}
}
