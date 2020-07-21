package sharedforeststore

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
)

//TagCounted supports both TaggedStore and CounterStore interfaces.
//The TaggedStore is backed by CounterStore and shares its counters.
type TagCounted struct {
	Counted
}

//NewTagCountedStore creates a new TagCounted from a transactional datastore.
func NewTagCountedStore(db datastore.TxnDatastore, opt *DatabaseOptions) *TagCounted {
	cs := NewCountedStore(db, opt)
	return &TagCounted{
		Counted: *cs,
	}
}

//txPutTag returns true if a new tag was added
func txPutTag(tx datastore.Txn, id cid.Cid, tag datastore.Key) (bool, error) {
	idtag := getTaggedKey(id, tag)
	_, err := tx.Get(idtag)
	if err != datastore.ErrNotFound {
		//tag already added, or some other error occurred
		return false, err
	}
	err = tx.Put(idtag, nil)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *TagCounted) PutTag(ctx context.Context, id cid.Cid, tag datastore.Key, bg BlockGetter) error {
	return c.txWarp(ctx, func(tx *Tx) error {
		put, err := txPutTag(tx.transaction, id, tag)
		if !put {
			return err
		}
		_, err = tx.increment(id, bg, c.opt.LinkDecoder)
		return err
	})
}

func (c *TagCounted) HasBlockTagged(ctx context.Context, id cid.Cid, tag datastore.Key) (bool, error) {
	return c.ds.Has(getTaggedKey(id, tag))
}

func (c *TagCounted) GetTags(ctx context.Context, id cid.Cid) ([]datastore.Key, error) {
	prefix := newKeyFromCid(id, tagSuffixKey)
	rs, err := c.ds.Query(query.Query{
		Filters:  []query.Filter{query.FilterKeyPrefix{Prefix: prefix.String()}},
		KeysOnly: true,
	})
	if err != nil {
		return nil, err
	}
	es, err := rs.Rest()
	if err != nil {
		return nil, err
	}
	ps := len(prefix.String())
	tags := make([]datastore.Key, len(es))
	for i, e := range es {
		tags[i] = datastore.RawKey(e.Key[ps:])
	}
	return tags, nil
}

func (c *TagCounted) RemoveTag(ctx context.Context, id cid.Cid, tag datastore.Key) error {
	tk := getTaggedKey(id, tag)
	return c.txWarp(ctx, func(tx *Tx) error {
		has, err := tx.transaction.Has(tk)
		if err != nil {
			return err
		}
		if !has {
			return nil
		}
		if err = tx.transaction.Delete(tk); err != nil {
			return err
		}
		_, err = tx.decrement(id, c.opt.LinkDecoder)
		return err
	})
}
