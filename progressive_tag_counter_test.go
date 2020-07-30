package sharedforeststore

import (
	"context"
	"fmt"
	"sync"
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
				ctx := gctx
				pm, _, err := store.ProgressiveIncrement(ctx, cids[c.node], getter)
				if err != nil {
					return err
				}
				r := ProgressReport{}
				if err := pm.CopyReport(&r); err != nil {
					return err
				}
				if c.tag == "B" {
					if err := pm.run(ctx); err != nil {
						return err
					}
				}
				if c.node > 1 {
					if err := store.ProgressiveContinue(ctx, cids[c.node], getter).Run(ctx); err != nil {
						return err
					}
				}
				if _, err := store.Decrement(ctx, cids[c.node]); err != nil {
					return err
				}
				pm = store.ProgressivePutTag(ctx, cids[c.node], datastore.NewKey(c.tag), getter)
				return pm.Run(ctx)
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
				if err := pm.Run(ctx); err != nil {
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

func BenchmarkPutRemoveProgrisveTag(b *testing.B) {
	cids, getter := setup(b)
	db, err := leveldb.NewDatastore("", nil)
	fatalIfErr(b, err)
	defer db.Close()
	store := NewProgressiveTagCountedStore(db, nil)
	ctx := context.Background()
	id := cids[1]
	tag := datastore.NewKey("tag")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fatalIfErr(b, store.ProgressivePutTag(ctx, id, tag, getter).Run(ctx))
		fatalIfErr(b, store.RemoveTag(ctx, id, tag))
	}
}

func BenchmarkPutRemoveProgressiveTag_P8(b *testing.B) {
	p := 8
	cids, getter := setup(b)
	db, err := leveldb.NewDatastore("", nil)
	fatalIfErr(b, err)
	defer db.Close()
	store := NewProgressiveTagCountedStore(db, nil)
	ctx := context.Background()
	id := cids[1]
	wg := sync.WaitGroup{}
	wg.Add(p)
	b.ResetTimer()
	bn := b.N
	for i := p; i > 0; i-- {
		n := bn / i
		bn -= n
		tag := datastore.NewKey(fmt.Sprint(i))
		go func() {
			defer wg.Done()
			for i := 0; i < n; i++ {
				fatalIfErr(b, store.ProgressivePutTag(ctx, id, tag, getter).Run(ctx))
				fatalIfErr(b, store.RemoveTag(ctx, id, tag))
			}
		}()
	}
	wg.Wait()
}
