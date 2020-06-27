package sharedforeststore

import (
	"bytes"
	"encoding/binary"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/multiformats/go-base32"
	"github.com/pkg/errors"
)

//newKeyFromCid creates a new key from a cid with suffixes.
//The encoding picked is the same as NewKeyFromBinary in go-ipfs-ds-help.
func newKeyFromCid(id cid.Cid, suffixKeys ...datastore.Key) datastore.Key {
	encoding := base32.RawStdEncoding
	keyLen := 1 + encoding.EncodedLen(id.ByteLen())
	for _, k := range suffixKeys {
		keyLen += len(k.String())
	}
	b := make([]byte, 1, keyLen)
	b[0] = '/'
	buf := bytes.NewBuffer(b)
	encoder := base32.NewEncoder(encoding, buf)
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

func getCount(db datastore.Read, id cid.Cid) (int64, counterKey, error) {
	key := getCounterKey(id)
	v, err := db.Get(datastore.Key(key))
	if err == datastore.ErrNotFound {
		return 0, key, nil
	}
	count, size := binary.Uvarint(v)
	if size != len(v) || count == 0 {
		return 0, key, errors.Errorf("corrupted metadata error: expected binary.Uvarint, but got `%x`", v)
	}
	return int64(count), key, nil
}

func setCount(db datastore.Write, v int64, k0 counterKey) error {
	k := datastore.Key(k0)
	if v == 0 {
		return db.Delete(k)
	} else if v < 0 {
		return errors.Errorf("can not set a count of less than 0 for key:%v, count:%v", k, v)
	}
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, uint64(v))
	return db.Put(k, buf[:n])
}

var dataSuffixKey = datastore.NewKey("/d")

func getDataKey(id cid.Cid) datastore.Key {
	return newKeyFromCid(id, dataSuffixKey)
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

func getTaggedKey(id cid.Cid, tag datastore.Key) datastore.Key {
	return newKeyFromCid(id, tagSuffixKey, tag)
}

var internalTagSuffixKey = datastore.NewKey("/i")

func getInternalTaggedKey(id cid.Cid, tag datastore.Key) datastore.Key {
	return newKeyFromCid(id, internalTagSuffixKey, tag)
}
