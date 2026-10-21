package oracle

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	oraTypes "github.com/sijms/go-ora/v3/types"
)

const InlineLOBUnits = 256 * 1024
const PreviewLOBUnits = 4096
const MaxExportLOBUnits = 2 * 1024 * 1024
const MaxRetainedLOBs = 1024

type lobReader interface {
	GetLocator() oraTypes.Locator
	GetLength() (int64, error)
	ReadBytesFromPos(context.Context, int64, int64) ([]byte, error)
	Free() error
	IsNil() bool
}
type lobReference struct {
	reader  lobReader
	kind    string
	charset int
	length  int64
	decode  func([]byte) string
}

func lobValue(value any) (lobReference, bool) {
	switch lob := value.(type) {
	case *oraTypes.Clob:
		ref := lobReference{reader: lob, kind: "CLOB"}
		if lob.UseNCharset {
			ref.kind = "NCLOB"
		}
		if lob.Conv != nil {
			ref.charset = lob.Conv.GetLangID()
			ref.decode = lob.Conv.Decode
		}
		return ref, true
	case *oraTypes.Blob:
		return lobReference{reader: lob, kind: "BLOB"}, true
	}
	return lobReference{}, false
}

func (s *Session) lobCall(call func() error) error {
	if s.conn == nil {
		return call()
	}
	return s.conn.Raw(func(_ any) error { return call() })
}

func (s *Session) freeLOB(ref lobReference) {
	if ref.reader.GetLocator().IsTemporary() {
		_ = s.lobCall(ref.reader.Free)
	}
}
func (s *Session) clearLOBs() {
	if len(s.lobs) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if s.conn != nil {
		stop, err := s.pageFetchContext(ctx)
		if err == nil {
			defer stop()
		}
	}
	for id, ref := range s.lobs {
		delete(s.lobs, id)
		if ctx.Err() == nil {
			s.freeLOB(ref)
		}
	}
}

func (s *Session) scanLOB(ctx context.Context, ref lobReference, retain bool) (Cell, error) {
	cell := Cell{Type: ref.kind}
	if ref.reader.IsNil() {
		cell.Null = true
		cell.Text = "NULL"
		return cell, nil
	}
	stop := func() {}
	if active, _ := ctx.Value(fetchWatchKey{}).(bool); !active && s.conn != nil {
		var err error
		stop, err = s.pageFetchContext(ctx)
		if err != nil {
			return cell, err
		}
	}
	err := s.lobCall(func() error { var err error; ref.length, err = ref.reader.GetLength(); return err })
	stop()
	if err != nil {
		return cell, err
	}
	cell.Size = ref.length
	if ref.length < 0 {
		s.freeLOB(ref)
		return cell, errors.New("negative LOB length")
	}
	limit := int64(InlineLOBUnits)
	if !retain {
		limit = MaxExportLOBUnits
	}
	if ref.length > limit {
		if !retain {
			s.freeLOB(ref)
			return cell, fmt.Errorf("%s cell exceeds the %d-unit export limit; save the individual cell to stream its full value", ref.kind, MaxExportLOBUnits)
		}
		limit = PreviewLOBUnits
	}
	var out strings.Builder
	_, units, err := s.writeLOB(ctx, &out, ref, limit)
	if err != nil {
		s.freeLOB(ref)
		return cell, err
	}
	cell.Text = out.String()
	if ref.kind == "BLOB" {
		cell.Text = hex.EncodeToString([]byte(cell.Text))
	}
	cell.Partial = units < ref.length
	if cell.Partial {
		if len(s.lobs) >= MaxRetainedLOBs {
			s.freeLOB(ref)
			return cell, errors.New("too many retained LOB values; narrow the query")
		}
		if s.lobs == nil {
			s.lobs = make(map[uint64]lobReference)
		}
		s.lobSequence++
		cell.LOBID = s.lobSequence
		s.lobs[cell.LOBID] = ref
	} else {
		s.freeLOB(ref)
	}
	return cell, nil
}

func (s *Session) writeLOB(ctx context.Context, w io.Writer, ref lobReference, limit int64) (written, units int64, err error) {
	goal := ref.length
	if limit > 0 {
		goal = min(goal, limit)
	}
	var carry []byte
	var order binary.ByteOrder = binary.BigEndian
	if ref.charset == 2002 {
		order = binary.LittleEndian
	}
	for units < goal || len(carry) > 0 {
		if err = ctx.Err(); err != nil {
			return
		}
		amount := min(int64(32768), goal-units)
		if amount <= 0 {
			amount = 1
		}
		amount = min(amount, ref.length-units)
		if amount <= 0 {
			err = errors.New("incomplete UTF-16 LOB value")
			return
		}
		var raw []byte
		readContext := ctx
		if active, _ := ctx.Value(fetchWatchKey{}).(bool); active {
			readContext = context.WithoutCancel(ctx)
		}
		err = s.lobCall(func() error { var e error; raw, e = ref.reader.ReadBytesFromPos(readContext, units, amount); return e })
		if err != nil {
			return
		}
		if len(raw) == 0 {
			err = io.ErrUnexpectedEOF
			return
		}
		var data []byte
		if ref.kind == "BLOB" {
			units += int64(len(raw))
			data = raw
		} else if ref.charset == 2000 || ref.charset == 2002 {
			if len(raw)%2 != 0 {
				err = errors.New("odd UTF-16 LOB byte count")
				return
			}
			units += int64(len(raw) / 2)
			data = append(carry, raw...)
			carry = nil
			if len(data) >= 2 {
				last := order.Uint16(data[len(data)-2:])
				if last >= 0xD800 && last <= 0xDBFF {
					carry = append([]byte(nil), data[len(data)-2:]...)
					data = data[:len(data)-2]
				}
			}
			chars := make([]uint16, len(data)/2)
			for i := range chars {
				chars[i] = order.Uint16(data[i*2:])
			}
			for i := 0; i < len(chars); i++ {
				if chars[i] >= 0xD800 && chars[i] <= 0xDBFF {
					if i+1 >= len(chars) || chars[i+1] < 0xDC00 || chars[i+1] > 0xDFFF {
						err = errors.New("invalid UTF-16 LOB surrogate")
						return
					}
					i++
				} else if chars[i] >= 0xDC00 && chars[i] <= 0xDFFF {
					err = errors.New("isolated UTF-16 LOB surrogate")
					return
				}
			}
			data = []byte(string(utf16.Decode(chars)))
		} else {
			if ref.decode == nil {
				err = errors.New("LOB character decoder is unavailable")
				return
			}
			text := ref.decode(raw)
			if text == "" || !utf8.ValidString(text) {
				err = errors.New("invalid LOB text encoding")
				return
			}
			for _, r := range text {
				units++
				if r > 0xFFFF {
					units++
				}
			}
			data = []byte(text)
		}
		if units > ref.length {
			err = errors.New("LOB length mismatch")
			return
		}
		if len(data) > 0 {
			var n int
			n, err = w.Write(data)
			written += int64(n)
			if err != nil {
				return
			}
			if n != len(data) {
				err = io.ErrShortWrite
				return
			}
		}
	}
	return
}

// WriteCell streams the original value, never a displayed ellipsis/hex dump.
// A LOB reference belongs to this session/execution and expires on new SQL or
// commit/rollback. Ordinary retained values need no live DB reference.
func (s *Session) WriteCell(ctx context.Context, w io.Writer, cell Cell) (int64, error) {
	if cell.Null {
		return 0, errors.New("NULL has no file content")
	}
	if cell.LOBID != 0 {
		ref, ok := s.lobs[cell.LOBID]
		if !ok {
			return 0, errors.New("LOB value has expired; run the SQL again")
		}
		n, _, err := s.writeLOB(ctx, w, ref, 0)
		return n, err
	}
	if cell.Partial {
		return 0, errors.New("full cell content is unavailable")
	}
	var source io.Reader = strings.NewReader(cell.Text)
	if cell.Type == "RAW" || cell.Type == "BLOB" {
		source = hex.NewDecoder(source)
	}
	buffer := make([]byte, 32768)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		n, err := source.Read(buffer)
		if n > 0 {
			written, e := w.Write(buffer[:n])
			total += int64(written)
			if e != nil {
				return total, e
			}
			if written != n {
				return total, io.ErrShortWrite
			}
		}
		if err == io.EOF {
			return total, nil
		}
		if err != nil {
			return total, err
		}
	}
}

func (s *Session) PreviewCell(ctx context.Context, cell Cell) (Cell, error) {
	if cell.LOBID == 0 {
		return cell, nil
	}
	ref, ok := s.lobs[cell.LOBID]
	if !ok {
		return cell, errors.New("LOB value has expired; run the SQL again")
	}
	var out strings.Builder
	_, units, err := s.writeLOB(ctx, &out, ref, InlineLOBUnits)
	if err != nil {
		return cell, err
	}
	cell.Text = out.String()
	if ref.kind == "BLOB" {
		cell.Text = hex.EncodeToString([]byte(cell.Text))
	}
	cell.Partial = units < ref.length
	return cell, nil
}
