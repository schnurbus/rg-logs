// Package logarchive detects ZIP combat-log uploads and streams the embedded
// log during ingest without writing an uncompressed copy to disk.
package logarchive

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// MaxUncompressedBytes caps a single log entry inside a ZIP (zip-bomb guard).
const MaxUncompressedBytes = 512 << 20 // 512 MiB

// IsZip reports whether data starts with a ZIP local-file header.
func IsZip(data []byte) bool {
	return len(data) >= 4 && data[0] == 'P' && data[1] == 'K' && data[2] == 0x03 && data[3] == 0x04
}

// LooksLikeZip reports whether the upload should be treated as a ZIP based on
// filename extension or magic bytes.
func LooksLikeZip(filename string, data []byte) bool {
	if IsZip(data) {
		return true
	}
	return strings.EqualFold(filepath.Ext(filename), ".zip")
}

// StorageMeta returns the object extension and Content-Type for storage.
// ZIPs use application/octet-stream so existing combat-logs buckets accept
// them without requiring allowed_mime_types to include application/zip.
func StorageMeta(filename string, data []byte) (ext, contentType string) {
	if LooksLikeZip(filename, data) {
		return ".zip", "application/octet-stream"
	}
	return ".txt", "text/plain"
}

// ValidateZip ensures data is a ZIP that contains at least one usable log file.
func ValidateZip(data []byte) error {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("invalid zip archive: %w", err)
	}
	entry, err := pickLogEntry(zr.File)
	if err != nil {
		return err
	}
	if entry.UncompressedSize64 > MaxUncompressedBytes {
		return fmt.Errorf("zip entry %q too large when uncompressed (max %d bytes)", entry.Name, MaxUncompressedBytes)
	}
	return nil
}

// Open opens a stored upload path for parsing. ZIP archives stream the chosen
// entry; plain files are opened directly.
func Open(path string) (io.ReadCloser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	isZip, err := fileLooksLikeZip(f, path)
	if err != nil {
		f.Close()
		return nil, err
	}
	if !isZip {
		return f, nil
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	zr, err := zip.NewReader(f, stat.Size())
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("open zip: %w", err)
	}
	entry, err := pickLogEntry(zr.File)
	if err != nil {
		f.Close()
		return nil, err
	}
	if entry.UncompressedSize64 > MaxUncompressedBytes {
		f.Close()
		return nil, fmt.Errorf("zip entry %q too large when uncompressed (max %d bytes)", entry.Name, MaxUncompressedBytes)
	}
	rc, err := entry.Open()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("open zip entry %q: %w", entry.Name, err)
	}
	return &zipEntryReader{Reader: rc, entry: rc, file: f}, nil
}

type zipEntryReader struct {
	io.Reader
	entry io.ReadCloser
	file  *os.File
}

func (z *zipEntryReader) Close() error {
	err1 := z.entry.Close()
	err2 := z.file.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

func fileLooksLikeZip(f *os.File, path string) (bool, error) {
	if strings.EqualFold(filepath.Ext(path), ".zip") {
		return true, nil
	}
	var hdr [4]byte
	n, err := f.Read(hdr[:])
	if err != nil && err != io.EOF {
		return false, err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return false, err
	}
	return IsZip(hdr[:n]), nil
}

func pickLogEntry(files []*zip.File) (*zip.File, error) {
	var candidates []*zip.File
	for _, f := range files {
		if f.FileInfo().IsDir() {
			continue
		}
		name := normalizeZipName(f.Name)
		if name == "" || shouldSkipZipPath(name) {
			continue
		}
		candidates = append(candidates, f)
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("zip contains no files")
	}

	txt := filterByExt(candidates, ".txt")
	pool := candidates
	if len(txt) > 0 {
		pool = txt
	}

	best := pool[0]
	bestScore := entryScore(best)
	for _, f := range pool[1:] {
		score := entryScore(f)
		if score > bestScore || (score == bestScore && f.UncompressedSize64 > best.UncompressedSize64) {
			best = f
			bestScore = score
		}
	}
	return best, nil
}

func filterByExt(files []*zip.File, ext string) []*zip.File {
	var out []*zip.File
	for _, f := range files {
		if strings.EqualFold(filepath.Ext(normalizeZipName(f.Name)), ext) {
			out = append(out, f)
		}
	}
	return out
}

func entryScore(f *zip.File) int {
	base := strings.ToLower(filepath.Base(normalizeZipName(f.Name)))
	score := 0
	if strings.Contains(base, "wowcombatlog") {
		score += 100
	}
	if strings.HasSuffix(base, ".txt") {
		score += 10
	}
	return score
}

func normalizeZipName(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	return strings.TrimPrefix(name, "/")
}

func shouldSkipZipPath(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasPrefix(lower, "__macosx/") {
		return true
	}
	base := filepath.Base(name)
	if strings.HasPrefix(base, ".") {
		return true
	}
	return false
}
