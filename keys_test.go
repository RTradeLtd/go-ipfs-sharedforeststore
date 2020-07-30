package sharedforeststore

import (
	"bytes"
	"reflect"
	"testing"
)

func Test_encodeDecodeCounter(t *testing.T) {
	tests := []struct {
		name    string
		bs      []byte
		count   int64
		meta    metadata
		wantErr bool
	}{{
		name:    "nil err",
		bs:      nil,
		wantErr: true,
	}, {
		name:    "zero err",
		bs:      []byte{0},
		wantErr: true,
	}, {
		name:    "bad varint err",
		bs:      []byte{0xff},
		wantErr: true,
	}, {
		name:    "bad meta err",
		bs:      []byte{1, 2},
		wantErr: true,
	}, {
		name:    "long meta err",
		bs:      []byte{1, 0, 0},
		wantErr: true,
	}, {
		name:    "1 complete",
		bs:      []byte{1},
		count:   1,
		meta:    metadata{Complete: true, HavePart: true},
		wantErr: false,
	}, {
		name:    "1 have part",
		bs:      []byte{1, 1},
		count:   1,
		meta:    metadata{Complete: false, HavePart: true},
		wantErr: false,
	}, {
		name:    "1 no part",
		bs:      []byte{1, 0},
		count:   1,
		meta:    metadata{Complete: false, HavePart: false},
		wantErr: false,
	}}
	for _, tt := range tests {
		t.Run("decode "+tt.name, func(t *testing.T) {
			gotC, gotM, err := decodeCounter(tt.bs)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeCounter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotC != tt.count {
				t.Errorf("decodeCounter() count = %v, want %v", gotC, tt.count)
			}
			if !reflect.DeepEqual(gotM, tt.meta) {
				t.Errorf("decodeCounter() meta = %v, want %v", gotM, tt.meta)
			}
		})
		t.Run("encode "+tt.name, func(t *testing.T) {
			if tt.wantErr {
				return
			}
			bs := tt.meta.encodeWithCount(tt.count)
			if !bytes.Equal(bs, tt.bs) {
				t.Errorf("encodeWithCount() got %x, want %x", bs, tt.bs)
			}
		})
	}
}
