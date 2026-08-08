package logarchive

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestIsZip(t *testing.T) {
	if IsZip([]byte("not a zip")) {
		t.Fatal("expected false")
	}
	if !IsZip([]byte{'P', 'K', 0x03, 0x04, 0x00}) {
		t.Fatal("expected true for local file header")
	}
}

func TestStorageMeta(t *testing.T) {
	ext, ct := StorageMeta("log.txt", []byte("plain"))
	if ext != ".txt" || ct != "text/plain" {
		t.Fatalf("got %q %q", ext, ct)
	}
	zipBytes := mustZip(t, map[string]string{"WoWCombatLog.txt": "line\n"})
	ext, ct = StorageMeta("raid.zip", zipBytes)
	if ext != ".zip" || ct != "application/octet-stream" {
		t.Fatalf("got %q %q", ext, ct)
	}
	ext, ct = StorageMeta("noext", zipBytes)
	if ext != ".zip" || ct != "application/octet-stream" {
		t.Fatalf("magic should force zip, got %q %q", ext, ct)
	}
}

func TestValidateZip(t *testing.T) {
	if err := ValidateZip([]byte("nope")); err == nil {
		t.Fatal("expected error")
	}
	data := mustZip(t, map[string]string{"WoWCombatLog.txt": "a\nb\n"})
	if err := ValidateZip(data); err != nil {
		t.Fatal(err)
	}
	empty := mustZip(t, nil)
	if err := ValidateZip(empty); err == nil {
		t.Fatal("expected empty zip error")
	}
}

func TestOpenPlain(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.txt")
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rc, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello\n" {
		t.Fatalf("got %q", got)
	}
}

func TestOpenZipStreamsPreferredEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "upload.zip")
	data := mustZip(t, map[string]string{
		"__MACOSX/._junk":     "meta",
		"notes.md":            "ignore",
		"other.txt":           "small",
		"WoWCombatLog.txt":    "combat-log-body\n",
		"nested/readme.txt":   "nested",
	})
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	rc, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "combat-log-body\n" {
		t.Fatalf("got %q", got)
	}
}

func TestOpenZipLargestTxtWhenNoWowName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "upload.zip")
	data := mustZip(t, map[string]string{
		"a.txt": "aa",
		"b.txt": "bbbbbbbb",
	})
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	rc, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "bbbbbbbb" {
		t.Fatalf("got %q", got)
	}
}

func TestPickLogEntrySkipsMacOS(t *testing.T) {
	data := mustZip(t, map[string]string{
		"__MACOSX/._WoWCombatLog.txt": "bad",
		"WoWCombatLog.txt":            "good\n",
	})
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	entry, err := pickLogEntry(zr.File)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(entry.Name) != "WoWCombatLog.txt" {
		t.Fatalf("got %q", entry.Name)
	}
}

func mustZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(w, body); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
