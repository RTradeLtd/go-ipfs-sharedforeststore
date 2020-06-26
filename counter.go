package sharedforeststore

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"go.uber.org/multierr"
)

type DatabaseOptions struct {
	LinkDecoder LinkDecoderFunc
}

type Counted struct {
	opt DatabaseOptions
	db  datastore.TxnDatastore
}

//CountedTx is a CounterStore where all actions are group in to a single transaction.
//
type CountedTx struct {
	context.Context
	tx datastore.Txn
}

//NewCountedStore creates a new Counted (implements CounterStore) from a transactional datastore.
func NewCountedStore(db datastore.TxnDatastore, opt *DatabaseOptions) *Counted {
	if opt == nil {
		opt = &DatabaseOptions{}
	}
	if opt.LinkDecoder == nil {
		opt.LinkDecoder = LinkDecoder
	}
	return &Counted{
		opt: *opt,
		db:  db,
	}
}

func (c *Counted) NewTransaction(ctx context.Context) (*CountedTx, error) {
	tx, err := c.db.NewTransaction(false)
	return &CountedTx{
		Context: ctx,
		tx:      tx,
	}, err
}

//TxWarp handles recourse management and commit retry for a transaction
func (c *Counted) TxWarp(ctx context.Context, f func(tx *CountedTx) error) error {
	tx, err := c.NewTransaction(ctx)
	if err != nil {
		return err
	}
	defer tx.tx.Discard()
	var commitError error
	for ctx.Err() == nil {
		if err := f(tx); err != nil {
			return multierr.Combine(err, commitError)
		}
		if commitError = tx.tx.Commit(); commitError == nil {
			return nil
		}
	}
	return multierr.Combine(err, commitError)
}

func (c *Counted) Increment(ctx context.Context, id cid.Cid, bg BlockGetter) (count uint64, err error) {
	err = c.TxWarp(ctx, func(tx *CountedTx) error {
		count, err = tx.Increment(id, bg, c.opt.LinkDecoder)
		return err
	})
	return
}

func (c *CountedTx) Increment(id cid.Cid, bg BlockGetter, ld LinkDecoderFunc) (uint64, error) {
	count, key, err := getCount(c.tx, id)
	if err != nil {
		return 0, err
	}
	count++
	if err := setCount(c.tx, count, key); err != nil {
		return 0, err
	}
	if count > 1 {
		return count, nil
	}
	data, err := bg.GetBlock(c, id)
	if err != nil {
		return 0, err
	}
	if err := setData(c.tx, id, data); err != nil {
		return 0, err
	}
	cids, err := ld(id, data)
	if err != nil {
		return 0, err
	}
	for _, linkedCid := range cids {
		if _, err := c.Increment(linkedCid, bg, ld); err != nil {
			return 0, err
		}
	}
	return count, nil
}
