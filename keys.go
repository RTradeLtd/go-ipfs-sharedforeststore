package sharedforeststore

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/pkg/errors"
)

//newKeyFromCid creates a new key from a cid with suffixes.
//The encoding picked is the same as NewKeyFromBinary in go-ipfs-ds-help.
func newKeyFromCid(id cid.Cid, suffixKeys ...datastore.Key) datastore.Key {
	encoding := base64.URLEncoding
	keyLen := 2 + encoding.EncodedLen(id.ByteLen())
	for _, k := range suffixKeys {
		keyLen += len(k.String())
	}
	b := make([]byte, 2, keyLen)
	b[0] = '/'
	b[1] = 85 //id for base64.URLEncoding
	buf := bytes.NewBuffer(b)
	encoder := base64.NewEncoder(encoding, buf)
	if _, err := id.WriteBytes(encoder); err != nil {
		panic(err) // error here should be impossible
	}
	if err := encoder.Close(); err != nil {
		panic(err) // error here should be impossible
	}
	for _, k := range suffixKeys {
		if _, err := buf.WriteString(k.String()); err != nil {
			panic(err) // error here should be impossible
		}
	}
	return datastore.RawKey(buf.String())
}

type readWriteStore interface {
	datastore.Read
	datastore.Write
}

var counterSuffixKey = datastore.NewKey("/c")

type counterKey datastore.Key

func getCounterKey(id cid.Cid) counterKey {
	return counterKey(newKeyFromCid(id, counterSuffixKey))
}

type metadata struct {
	Complete bool
	HavePart bool
}

func decodeCounter(bs []byte) (int64, metadata, error) {
	c, size := binary.Uvarint(bs)
	count := int64(c)
	if count <= 0 {
		return 0, metadata{}, errors.Errorf("corrupted metadata error: count less than 1, from raw `%x`", bs)
	}
	switch len(bs) {
	case size:
		return count, metadata{Complete: true, HavePart: true}, nil
	case size + 1:
		if bs[size] > 1 {
			return 0, metadata{}, errors.Errorf("corrupted metadata error: meta > 1, from raw `%x`", bs)
		}
		return count, metadata{Complete: false, HavePart: bs[size] == 1}, nil
	default:
		return 0, metadata{}, errors.Errorf("corrupted metadata error: length too long, from raw `%x`", bs)
	}
}

func (m metadata) encodeWithCount(c int64) []byte {
	padding := 0
	if !m.Complete {
		padding = 1
	}
	buf := make([]byte, binary.MaxVarintLen64+padding)
	n := binary.PutUvarint(buf, uint64(c))
	if m.HavePart && !m.Complete {
		buf[n] = 1
	}
	return buf[:n+padding]
}

func getCount(db datastore.Read, id cid.Cid) (int64, metadata, counterKey, error) {
	key := getCounterKey(id)
	v, err := db.Get(datastore.Key(key))
	if err == datastore.ErrNotFound {
		return 0, metadata{}, key, nil
	}
	count, meta, err := decodeCounter(v)
	return count, meta, key, err
}

func setCount(db datastore.Write, k0 counterKey, v int64, meta metadata) error {
	k := datastore.Key(k0)
	if v == 0 {
		return db.Delete(k)
	}
	if v < 0 {
		return errors.Errorf("can not set a count of less than 0 for key:%v, count:%v", k, v)
	}
	return db.Put(k, meta.encodeWithCount(v))
}

var dataSuffixKey = datastore.NewKey("/d")

func getDataKey(id cid.Cid) datastore.Key {
	return newKeyFromCid(id, dataSuffixKey)
}

func dataKeyToCid(s string) (cid.Cid, error) {
	if len(s) < 4 {
		return cid.Cid{}, errors.Errorf("key:%v is too short to contain cid", s)
	}
	return cid.Decode(s[1 : len(s)-len(dataSuffixKey.String())])
}

func setData(db datastore.Write, id cid.Cid, data []byte) error {
	return db.Put(getDataKey(id), data)
}

func deleteData(db readWriteStore, id cid.Cid) ([]byte, error) {
	key := getDataKey(id)
	data, err := db.Get(key)
	if err != nil {
		return nil, err
	}
	return data, db.Delete(key)
}

var tagSuffixKey = datastore.NewKey("/t")

func getTagKey(id cid.Cid, tag datastore.Key) datastore.Key {
	return newKeyFromCid(id, tagSuffixKey, tag)
}

var internalTagSuffixKey = datastore.NewKey("/i")

func getInternalTagKey(id cid.Cid, tag datastore.Key) datastore.Key {
	return newKeyFromCid(id, internalTagSuffixKey, tag)
}

var _ = getInternalTagKey //block used warnings during development
