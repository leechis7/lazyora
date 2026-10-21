package oracle

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	oraTypes "github.com/sijms/go-ora/v3/types"
	"io"
	"strings"
	"testing"
	"unicode/utf16"
)

type memoryLOB struct {
	data  []byte
	char  bool
	null  bool
	reads int
}

func (m *memoryLOB) GetLocator() oraTypes.Locator { return nil }
func (m *memoryLOB) GetLength() (int64, error) {
	if m.char {
		return int64(len(m.data) / 2), nil
	}
	return int64(len(m.data)), nil
}
func (m *memoryLOB) IsNil() bool { return m.null }
func (m *memoryLOB) Free() error { return nil }
func (m *memoryLOB) ReadBytesFromPos(ctx context.Context, off, n int64) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.reads++
	if m.char {
		off *= 2
		n *= 2
	}
	return m.data[off:min(off+n, int64(len(m.data)))], nil
}
func utf16LOB(text string) *memoryLOB {
	units := utf16.Encode([]rune(text))
	data := make([]byte, 2*len(units))
	for i, v := range units {
		binary.BigEndian.PutUint16(data[2*i:], v)
	}
	return &memoryLOB{data: data, char: true}
}
func TestLargeLOBPreviewAndExactStreaming(t *testing.T) {
	text := strings.Repeat("x", 32767) + "👩한글\n" + strings.Repeat("한", InlineLOBUnits)
	blob := bytes.Repeat([]byte{0, 1, 255, 128}, InlineLOBUnits/4+1)
	for _, tc := range []struct {
		kind    string
		lob     *memoryLOB
		want    []byte
		charset int
	}{{"CLOB", utf16LOB(text), []byte(text), 2000}, {"BLOB", &memoryLOB{data: blob}, blob, 0}} {
		t.Run(tc.kind, func(t *testing.T) {
			s := &Session{}
			ctx := context.Background()
			cell, err := s.scanLOB(ctx, lobReference{reader: tc.lob, kind: tc.kind, charset: tc.charset}, true)
			if err != nil || !cell.Partial || cell.LOBID == 0 {
				t.Fatalf("preview: %+v %v", cell, err)
			}
			var out bytes.Buffer
			n, err := s.WriteCell(ctx, &out, cell)
			if err != nil || n != int64(len(tc.want)) || !bytes.Equal(out.Bytes(), tc.want) {
				t.Fatalf("stream mismatch %d %v", n, err)
			}
			detail, err := s.PreviewCell(ctx, cell)
			if err != nil || !detail.Partial || len(detail.Text) <= len(cell.Text) {
				t.Fatalf("detail: %v", err)
			}
			s.clearLOBs()
			if _, err = s.WriteCell(ctx, io.Discard, cell); err == nil {
				t.Fatal("expired locator reused")
			}
		})
	}
}
func TestLOBNullEmptyAndBinaryRetained(t *testing.T) {
	s := &Session{}
	for _, null := range []bool{false, true} {
		c, err := s.scanLOB(context.Background(), lobReference{reader: &memoryLOB{null: null}, kind: "BLOB"}, true)
		if err != nil || c.Null != null || c.Partial || c.LOBID != 0 {
			t.Fatalf("empty/null: %+v %v", c, err)
		}
	}
	raw := []byte{0, 255, 1}
	var out bytes.Buffer
	if _, err := s.WriteCell(context.Background(), &out, Cell{Type: "RAW", Text: hex.EncodeToString(raw)}); err != nil || !bytes.Equal(out.Bytes(), raw) {
		t.Fatal("binary was saved as displayed hex")
	}
	if _, err := s.WriteCell(context.Background(), io.Discard, Cell{Null: true}); err == nil {
		t.Fatal("NULL saved as empty")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.WriteCell(ctx, io.Discard, Cell{Text: "data"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

type shortLOBWriter struct{}

func (shortLOBWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }
func TestLOBRejectsInvalidSurrogatesShortWritesAndExportOverflow(t *testing.T) {
	s := &Session{}
	ctx := context.Background()
	for _, data := range [][]byte{{0xd8, 0}, {0xdc, 0}, {0}} {
		_, _, err := s.writeLOB(ctx, io.Discard, lobReference{reader: &memoryLOB{data: data, char: true}, kind: "CLOB", charset: 2000, length: 1}, 0)
		if err == nil {
			t.Fatal("invalid UTF16 accepted")
		}
	}
	ref := lobReference{reader: &memoryLOB{data: []byte{1, 2}}, kind: "BLOB", length: 2}
	if _, _, err := s.writeLOB(ctx, shortLOBWriter{}, ref, 0); !errors.Is(err, io.ErrShortWrite) {
		t.Fatal(err)
	}
	large := &memoryLOB{data: make([]byte, MaxExportLOBUnits+1)}
	if _, err := s.scanLOB(ctx, lobReference{reader: large, kind: "BLOB"}, false); err == nil || large.reads != 0 {
		t.Fatal("oversized export must fail before allocation")
	}
}
