package sharedforeststore

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
)

//ProgressiveTagCounted supports both ProgressiveTaggedStore and ProgressiveCounterStore interfaces.
//It is backed by CounterStore and shares its counters.
type ProgressiveTagCounted struct {
	TagCounted
	ProgressiveCounted
}

//NewProgressiveTagCountedStore creates a new ProgressiveTagCounted from a transactional datastore.
func NewProgressiveTagCountedStore(db datastore.TxnDatastore, opt *DatabaseOptions) *ProgressiveTagCounted {
	cs := NewTagCountedStore(db, opt)
	return &ProgressiveTagCounted{
		TagCounted: *cs,
		ProgressiveCounted: ProgressiveCounted{
			Counted: cs.Counted,
		},
	}
}

func (c *ProgressiveTagCounted) ProgressivePutTag(ctx context.Context, id cid.Cid, tag datastore.Key, bg BlockGetter) ProgressManager {
	var meta metadata
	err := c.TagCounted.txWarp(ctx, func(tx *Tx) (err error) {
		put, err := txPutTag(tx.transaction, id, tag)
		if !put {
			return err
		}
		var count int64
		var key counterKey
		count, meta, key, err = getCount(tx.transaction, id)
		if err != nil {
			return err
		}
		count++
		return setCount(tx.transaction, key, count, meta)
	})
	if err != nil {
		return &StoreProgressManager{err: err}
	}
	if meta.Complete {
		return nil
	}
	return c.ProgressiveContinue(ctx, id, bg)
}
