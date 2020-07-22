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
	run        func(context.Context) error
	report     ProgressReport
	reportLock sync.RWMutex
}

func (m *StoreProgressManager) Run(ctx context.Context) error {
	if m == nil {
		return nil
	}
	if m.err != nil {
		return m.err
	}
	m.err = ErrRunOnce
	return m.run(ctx)
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

func (m *StoreProgressManager) updateReport(f func(r *ProgressReport)) {
	m.reportLock.Lock()
	defer m.reportLock.Unlock()
	f(&(m.report))
}

func (c *ProgressiveCounted) ProgressiveContinue(ctx context.Context, id cid.Cid, bg BlockGetter) *StoreProgressManager {
	m := &StoreProgressManager{}
	var r func(id cid.Cid) error // r is called recursively
	r = func(id cid.Cid) error {
		for {
			cids, err := c.progressTx(ctx, id, bg, m)
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
	m.run = func(ctx2 context.Context) error {
		ctx = ctx2
		return r(id)
	}
	return m
}

func (c *ProgressiveCounted) progressTx(ctx context.Context, id cid.Cid, bg BlockGetter, m *StoreProgressManager) ([]cid.Cid, error) {
	var cids []cid.Cid
	var size uint64
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
		var allLinks []cid.Cid
		if allLinks, size, err = c.opt.LinkDecoder(id, data); err != nil {
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
	m.updateReport(func(r *ProgressReport) {
		if !r.initalized {
			r.initalized = true
			if size != 0 {
				r.KnownBytes = size
			}
		}
		if len(cids) == 0 {
			r.HaveBytes = size
		}
	})
	return cids, nil
}

var ErrSizeNotSupported = errors.New("size not supported")

func (c *ProgressiveCounted) GetProgressReport(ctx context.Context, id cid.Cid, r *ProgressReport) error {
	*r = ProgressReport{initalized: true} //reset
	data, err := c.GetBlock(ctx, id)
	if err != nil {
		return err
	}
	_, size, err := c.opt.LinkDecoder(id, data)
	if err != nil {
		return err
	}
	if size == 0 {
		return ErrSizeNotSupported
	}
	r.KnownBytes = size

	var sum func(cid.Cid) (uint64, error)
	sum = func(id cid.Cid) (uint64, error) {
		_, meta, _, err := getCount(c.ds, id)
		if err != nil {
			return 0, err
		}
		if !meta.HavePart {
			return 0, nil
		}
		data, err := c.GetBlock(ctx, id)
		if err != nil {
			return 0, err
		}
		cids, size, err := c.opt.LinkDecoder(id, data)
		if err != nil {
			return 0, err
		}
		if meta.Complete {
			return size, nil
		} else {
			size = 0
			for _, id := range cids {
				n, _ := sum(id) //ignore error here so we add as much as possible
				size += n
			}
			return size, nil
		}
	}
	r.HaveBytes, err = sum(id)
	return err
}
