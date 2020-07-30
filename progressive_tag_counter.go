package sharedforeststore

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
)

//ProgressiveTagCounted supports both ProgressiveTagStore and ProgressiveCounterStore interfaces.
//It is backed by CounterStore and shares its counters.
type ProgressiveTagCounted struct {
	TagCounted
}

var _ ProgressiveTagCounterStore = (*ProgressiveTagCounted)(nil)

//NewProgressiveTagCountedStore creates a new ProgressiveTagCounted from a transactional datastore.
func NewProgressiveTagCountedStore(db datastore.TxnDatastore, opt *DatabaseOptions) *ProgressiveTagCounted {
	cs := NewTagCountedStore(db, opt)
	return &ProgressiveTagCounted{
		TagCounted: *cs,
	}
}

func (c *ProgressiveTagCounted) ProgressiveIncrement(ctx context.Context, id cid.Cid, bg BlockGetter) (ProgressManager, int64, error) {
	return (&ProgressiveCounted{c.Counted}).ProgressiveIncrement(ctx, id, bg)
}

func (c *ProgressiveTagCounted) ProgressiveContinue(ctx context.Context, id cid.Cid, bg BlockGetter) ProgressManager {
	return (&ProgressiveCounted{c.Counted}).ProgressiveContinue(ctx, id, bg)
}

func (c *ProgressiveTagCounted) GetProgressReport(ctx context.Context, id cid.Cid, r *ProgressReport) error {
	return (&ProgressiveCounted{c.Counted}).GetProgressReport(ctx, id, r)
}

func (c *ProgressiveTagCounted) ProgressivePutTag(ctx context.Context, id cid.Cid, tag datastore.Key, bg BlockGetter) ProgressManager {
	var meta metadata
	err := c.txWarp(ctx, func(tx *Tx) (err error) {
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
		return ProgressCompleted
	}
	return c.ProgressiveContinue(ctx, id, bg)
}
