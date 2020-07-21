package sharedforeststore

import (
	"context"
	"testing"

	"github.com/ipfs/go-datastore"
	leveldb "github.com/ipfs/go-ds-leveldb"
	"golang.org/x/sync/errgroup"
)

func TestProgressiveTagCounter(t *testing.T) {

	cids, getter := setup(t)
	db, err := leveldb.NewDatastore("", nil)
	fatalIfErr(t, err)
	defer db.Close()
	store := NewProgressiveTagCountedStore(db, nil)
	ctx := context.Background()

	var tagCases = []tagTestCase{
		{node: 0, tag: "A"},
		{node: 1, tag: "A"},
		{node: 2, tag: "A"},
		{node: 3, tag: "A"},
		{node: 0, tag: "B"},
		{node: 1, tag: "B"},
		{node: 2, tag: "B"},
		{node: 3, tag: "B"},
	}

	for i := 0; i < 5; i++ {
		group, gctx := errgroup.WithContext(ctx)
		for _, c := range tagCases {
			c := c
			group.Go(func() error {
				pm := store.ProgressivePutTag(gctx, cids[c.node], datastore.NewKey(c.tag), getter)
				return pm.Run()
			})
		}
		fatalIfErr(t, group.Wait())
		checkFullStoreByIterator(t, ctx, cids, store)
		group, gctx = errgroup.WithContext(ctx)
		for _, c := range tagCases {
			c := c
			group.Go(func() error {
				return store.RemoveTag(gctx, cids[c.node], datastore.NewKey(c.tag))
			})
		}
		fatalIfErr(t, group.Wait())
		//all counts should now be zero
		checkCounts(t, ctx, make([]int64, len(cids)), cids, store)
		//no blocks from store should be left
		checkFullStoreByIterator(t, ctx, nil, store)
	}

	group, gctx := errgroup.WithContext(ctx)
	for _, c := range tagCases {
		c := c
		group.Go(func() error {
			ctx := gctx
			for i := 0; i < 20; i++ {
				pm := store.ProgressivePutTag(ctx, cids[c.node], datastore.NewKey(c.tag), getter)
				if err := pm.Run(); err != nil {
					t.Error(err)
					return err
				}
				if err := store.RemoveTag(ctx, cids[c.node], datastore.NewKey(c.tag)); err != nil {
					t.Error(err)
					return err
				}
			}
			return nil
		})
	}
	fatalIfErr(t, group.Wait())
	//all counts should now be zero
	checkCounts(t, ctx, make([]int64, len(cids)), cids, store)
	//no blocks from store should be left
	checkFullStoreByIterator(t, ctx, nil, store)
}
