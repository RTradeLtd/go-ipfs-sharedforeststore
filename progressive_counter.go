package sharedforeststore

import (
	"context"
	"sync"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/pkg/errors"
)

type ProgressiveCounted struct {
	Counted
}

//NewProgressiveCountedStore creates a new ProgressiveCounted (implements ProgressiveCounterStore) from a transactional datastore.
func NewProgressiveCountedStore(ds datastore.TxnDatastore, opt *DatabaseOptions) *ProgressiveCounted {
	return &ProgressiveCounted{
		Counted: *NewCountedStore(ds, opt),
	}
}

func (c *ProgressiveCounted) ProgressiveIncrement(ctx context.Context, id cid.Cid, bg BlockGetter) (*StoreProgressManager, int64, error) {
	var count int64
	var meta metadata
	err := c.txWarp(ctx, func(tx *Tx) (err error) {
		var key counterKey
		count, meta, key, err = getCount(tx.transaction, id)
		if err != nil {
			return err
		}
		count++
		return setCount(tx.transaction, key, count, meta)
	})
	if err != nil {
		return nil, 0, err
	}
	if meta.Complete {
		return nil, count, nil
	}
	return c.ProgressiveContinue(ctx, id, bg), count, nil
}

var ErrProgressReverted = errors.New("progress was reverted by an other action")
var ErrRunOnce = errors.New("progress can only run once")

//StoreProgressManager implements ProgressManager
type StoreProgressManager struct {
	err        error
	run        func() error
	report     ProgressReport
	reportLock sync.RWMutex
}

func (m *StoreProgressManager) Run() error {
	if m == nil {
		return nil
	}
	if m.err != nil {
		return m.err
	}
	m.err = ErrRunOnce
	return m.run()
}

func (m *StoreProgressManager) CopyReport(r *ProgressReport) error {
	if m == nil {
		return nil
	}
	m.reportLock.RLock()
	defer m.reportLock.RUnlock()
	*r = m.report
	return nil
}

func (c *ProgressiveCounted) ProgressiveContinue(ctx context.Context, id cid.Cid, bg BlockGetter) *StoreProgressManager {
	m := &StoreProgressManager{}
	var r func(id cid.Cid) error
	r = func(id cid.Cid) error {
		for {
			cids, err := c.progressTx(ctx, id, bg)
			if len(cids) == 0 || err != nil {
				return err
			}
			for _, id := range cids {
				if err := r(id); err != nil {
					return err
				}
			}
		}
	}
	m.run = func() error {
		return r(id)
	}
	//TODO: update report as we progress
	return m
}

func (c *ProgressiveCounted) progressTx(ctx context.Context, id cid.Cid, bg BlockGetter) ([]cid.Cid, error) {
	var cids []cid.Cid
	err := c.txWarp(ctx, func(tx *Tx) error {
		count, meta, key, err := getCount(tx.transaction, id)
		if err != nil {
			return err
		}
		if count == 0 {
			return ErrProgressReverted
		}
		if meta.Complete {
			cids = nil
			return nil
		}
		getter := bg
		if meta.HavePart {
			getter = c
		}
		data, err := getter.GetBlock(ctx, id)
		if err != nil {
			return err
		}
		if !meta.HavePart {
			if err := tx.transaction.Put(getDataKey(id), data); err != nil {
				return err
			}
		}
		allLinks, err := c.opt.LinkDecoder(id, data)
		if err != nil {
			return err
		}
		if cids == nil {
			cids = make([]cid.Cid, 0, len(allLinks))
		} else {
			cids = cids[:0]
		}
		increment := !meta.HavePart
		for _, link := range allLinks {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			count, meta, key, err := getCount(tx.transaction, link)
			if err != nil {
				return err
			}
			if increment {
				count++
				if err := setCount(tx.transaction, key, count, meta); err != nil {
					return err
				}
			}
			if !meta.Complete {
				cids = append(cids, link)
			}
		}
		if len(cids) == 0 {
			if err := setCount(tx.transaction, key, count, metadata{Complete: true, HavePart: true}); err != nil {
				return err
			}
		} else if !meta.HavePart {
			if err := setCount(tx.transaction, key, count, metadata{Complete: false, HavePart: true}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return cids, nil
}

var ErrNotImplemented = errors.New("not implemented")

func (c *ProgressiveCounted) GetProgressReport(context.Context, cid.Cid, *ProgressReport) error {
	return ErrNotImplemented
}
