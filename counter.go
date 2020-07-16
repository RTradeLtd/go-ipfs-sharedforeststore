package sharedforeststore

import (
	"context"
	"io"
	"strings"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"go.uber.org/multierr"
)

type DatabaseOptions struct {
	LinkDecoder LinkDecoderFunc
}

type Counted struct {
	opt DatabaseOptions
	ds  datastore.TxnDatastore
}

//Tx is a datastore transaction where all actions are group in to a single transaction.
type Tx struct {
	context.Context
	transaction datastore.Txn
}

//NewCountedStore creates a new Counted (implements CounterStore) from a transactional datastore.
func NewCountedStore(ds datastore.TxnDatastore, opt *DatabaseOptions) *Counted {
	if opt == nil {
		opt = &DatabaseOptions{}
	}
	if opt.LinkDecoder == nil {
		opt.LinkDecoder = LinkDecoder
	}
	return &Counted{
		opt: *opt,
		ds:  ds,
	}
}

func (c *Counted) NewTransaction(ctx context.Context) (*Tx, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	tx, err := c.ds.NewTransaction(false)
	if err != nil {
		return nil, err
	}
	return &Tx{
		Context:     ctx,
		transaction: tx,
	}, err
}

//TxWarp handles resource management and commit retry for a transaction
func (c *Counted) TxWarp(ctx context.Context, f func(tx *Tx) error) error {
	tx, err := c.NewTransaction(ctx)
	if err != nil {
		return err
	}
	defer tx.transaction.Discard()
	var commitError error
	for {
		if err := f(tx); err != nil {
			return multierr.Combine(err, commitError)
		}
		if commitError = tx.transaction.Commit(); commitError == nil {
			return nil
		}
		if err := tx.Reset(c.ds); err != nil {
			return multierr.Combine(err, commitError)
		}
	}
}

func (c *Tx) Reset(db datastore.TxnDatastore) (err error) {
	if err := c.Err(); err != nil {
		return err
	}
	c.transaction.Discard()
	c.transaction, err = db.NewTransaction(false)
	return err
}

func (c *Counted) Increment(ctx context.Context, id cid.Cid, bg BlockGetter) (count int64, err error) {
	err = c.TxWarp(ctx, func(tx *Tx) error {
		count, err = tx.Increment(id, bg, c.opt.LinkDecoder)
		return err
	})
	return
}

func (c *Tx) Increment(id cid.Cid, bg BlockGetter, ld LinkDecoderFunc) (int64, error) {
	if err := c.Err(); err != nil {
		return 0, err
	}
	count, key, err := getCount(c.transaction, id)
	if err != nil {
		return 0, err
	}
	count++
	if err := setCount(c.transaction, count, key); err != nil {
		return 0, err
	}
	if count > 1 {
		return count, nil
	}
	data, err := bg.GetBlock(c, id)
	if err != nil {
		return 0, err
	}
	if err := setData(c.transaction, id, data); err != nil {
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

func (c *Counted) GetCount(ctx context.Context, id cid.Cid) (count int64, err error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	count, _, err = getCount(c.ds, id)
	return count, err
}

func (c *Counted) Decrement(ctx context.Context, id cid.Cid) (count int64, err error) {
	err = c.TxWarp(ctx, func(tx *Tx) error {
		count, err = tx.Decrement(id, c.opt.LinkDecoder)
		return err
	})
	return
}

func (c *Tx) Decrement(id cid.Cid, ld LinkDecoderFunc) (int64, error) {
	if err := c.Err(); err != nil {
		return 0, err
	}
	count, key, err := getCount(c.transaction, id)
	if err != nil {
		return 0, err
	}
	count--
	if count < 0 {
		return count, nil
	}
	if err := setCount(c.transaction, count, key); err != nil {
		return 0, err
	}
	if count > 0 {
		return count, nil
	}
	data, err := deleteData(c.transaction, id)
	if err != nil {
		return 0, err
	}
	cids, err := ld(id, data)
	if err != nil {
		return 0, err
	}
	for _, linkedCid := range cids {
		if _, err := c.Decrement(linkedCid, ld); err != nil {
			return 0, err
		}
	}
	return count, nil
}

func (c *Counted) GetBlock(ctx context.Context, id cid.Cid) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return c.ds.Get(getDataKey(id))
}

func (c *Counted) GetBlockSize(ctx context.Context, id cid.Cid) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return c.ds.GetSize(getDataKey(id))
}

type ckiter struct {
	rs  query.Results
	err error
}

func (c *ckiter) NextCid() (cid.Cid, error) {
	if c.err != nil {
		return cid.Undef, c.err
	}
	r, more := c.rs.NextSync()
	if r.Error != nil {
		return cid.Undef, r.Error
	}
	if !more {
		c.err = io.EOF // will return EOF on the next call
		if r.Key == "" {
			return cid.Undef, c.err
		}
	}
	return dataKeyToCid(r.Key) //should never error here since that is filtered
}

func (c *ckiter) Filter(e query.Entry) bool {
	if !strings.HasSuffix(e.Key, dataSuffixKey.String()) {
		return false
	}
	_, err := dataKeyToCid(e.Key)
	return err == nil
}

func (c *ckiter) Close() error {
	if c.rs == nil {
		return c.err
	}
	return c.rs.Close()
}

func (c *Counted) KeysIterator(prefix string) CidIterator {
	it := &ckiter{}
	it.rs, it.err = c.ds.Query(query.Query{
		Filters:  []query.Filter{it},
		KeysOnly: true,
	})
	return it
}
