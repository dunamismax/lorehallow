// Package woff wraps a TrueType font in the WOFF 1.0 container: the same
// tables, each compressed with zlib. Every browser reads it, and zlib is in
// the standard library (WOFF 2 needs Brotli, which isn't).
package woff

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
)

// FromSFNT converts a TrueType (sfnt) font to WOFF. The output depends only
// on the input.
func FromSFNT(ttf []byte) ([]byte, error) {
	be := binary.BigEndian
	if len(ttf) < 12 {
		return nil, errors.New("woff: font too short")
	}
	flavor := be.Uint32(ttf)
	if flavor != 0x00010000 && flavor != 0x4F54544F { // TrueType or 'OTTO'
		return nil, fmt.Errorf("woff: not an sfnt font (flavor %#x)", flavor)
	}
	n := int(be.Uint16(ttf[4:]))
	if len(ttf) < 12+16*n {
		return nil, errors.New("woff: table directory truncated")
	}
	type table struct {
		tag        [4]byte
		checksum   uint32
		orig, comp []byte
	}
	tables := make([]table, n)
	total := 12 + 16*n
	for i := range n {
		rec := ttf[12+16*i:]
		off, length := int(be.Uint32(rec[8:])), int(be.Uint32(rec[12:]))
		if off < 0 || length < 0 || off+length > len(ttf) {
			return nil, fmt.Errorf("woff: table %q out of bounds", rec[:4])
		}
		t := table{checksum: be.Uint32(rec[4:]), orig: ttf[off : off+length]}
		copy(t.tag[:], rec[:4])
		var z bytes.Buffer
		w, _ := zlib.NewWriterLevel(&z, zlib.BestCompression)
		_, _ = w.Write(t.orig)
		if err := w.Close(); err != nil {
			return nil, err
		}
		t.comp = t.orig
		if z.Len() < len(t.orig) {
			t.comp = z.Bytes()
		}
		tables[i] = t
		total += (length + 3) &^ 3
	}
	hdrLen := 44 + 20*n
	var data bytes.Buffer
	dir := make([]byte, 20*n)
	for i, t := range tables {
		rec := dir[20*i:]
		copy(rec, t.tag[:])
		be.PutUint32(rec[4:], uint32(hdrLen+data.Len()))
		be.PutUint32(rec[8:], uint32(len(t.comp)))
		be.PutUint32(rec[12:], uint32(len(t.orig)))
		be.PutUint32(rec[16:], t.checksum)
		data.Write(t.comp)
		for data.Len()%4 != 0 {
			data.WriteByte(0)
		}
	}
	hdr := make([]byte, 44)
	copy(hdr, "wOFF")
	be.PutUint32(hdr[4:], flavor)
	be.PutUint32(hdr[8:], uint32(hdrLen+data.Len()))
	be.PutUint16(hdr[12:], uint16(n))
	be.PutUint32(hdr[16:], uint32(total))
	be.PutUint16(hdr[20:], 1) // font version 1.0
	out := append(hdr, dir...)
	return append(out, data.Bytes()...), nil
}
