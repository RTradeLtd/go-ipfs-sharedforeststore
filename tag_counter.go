package sharedforeststore

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
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

func (c *TagCounted) PutTag(ctx context.Context, tag datastore.Key, id cid.Cid, bg BlockGetter) error {
	return c.TxWarp(ctx, func(tx *Tx) error {
		idtag := getTaggedKey(id, tag)
		_, err := tx.transaction.Get(idtag)
		if err != datastore.ErrNotFound {
			return err
		}
		err = tx.transaction.Put(idtag, nil)
		if err != nil {
			return err
		}
		_, err = tx.Increment(id, bg, c.opt.LinkDecoder)
		return err
	})
}
