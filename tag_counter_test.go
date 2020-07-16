package sharedforeststore

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/ipfs/go-datastore"
	leveldb "github.com/ipfs/go-ds-leveldb"
)

func TestTagCounter(t *testing.T) {
	t.Parallel()

	type testCase struct {
		node   int
		tag    string
		counts []int64
	}
	cases := []testCase{
		{node: 0, tag: "A", counts: []int64{1, 0, 0, 1, 0, 1}},
		{node: 1, tag: "A", counts: []int64{1, 1, 0, 2, 1, 3}},
		{node: 2, tag: "B", counts: []int64{1, 1, 1, 2, 2, 3}},
		{node: 3, tag: "C", counts: []int64{1, 1, 1, 3, 2, 3}},
		{node: 0, tag: "A", counts: []int64{1, 1, 1, 3, 2, 3}},
		{node: 0, tag: "B", counts: []int64{2, 1, 1, 3, 2, 3}},
	}

	cids, getter := setup(t)
	db, err := leveldb.NewDatastore("", nil)
	fatalIfErr(t, err)
	store := NewTagCountedStore(db, nil)
	ctx := context.Background()

	for _, c := range cases {
		fatalIfErr(t, store.PutTag(ctx, cids[c.node], datastore.NewKey(c.tag), getter))
		checkCounts(t, ctx, c.counts, cids, store)
	}

	checkFullStoreByIterator(t, ctx, cids, store)

	for _, c := range cases {
		fatalIfErr(t, store.RemoveTag(ctx, cids[c.node], datastore.NewKey(c.tag)))
	}
	//all counts should now be zero
	checkCounts(t, ctx, make([]int64, len(cids)), cids, store)
	//no blocks from store should be left
	checkFullStoreByIterator(t, ctx, nil, store)
}

func BenchmarkPutTag(b *testing.B) {
	cids, getter := setup(b)
	db, err := leveldb.NewDatastore("", nil)
	fatalIfErr(b, err)
	store := NewTagCountedStore(db, nil)
	ctx := context.Background()
	id := cids[1]
	tag := datastore.NewKey("tag")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fatalIfErr(b, store.PutTag(ctx, id, tag, getter))
	}
}

func BenchmarkPutTag_P8(b *testing.B) {
	p := 8
	cids, getter := setup(b)
	db, err := leveldb.NewDatastore("", nil)
	fatalIfErr(b, err)
	store := NewTagCountedStore(db, nil)
	ctx := context.Background()
	tag := datastore.NewKey("tag")
	id := cids[1]
	wg := sync.WaitGroup{}
	wg.Add(p)
	b.ResetTimer()
	//start p go-rountines
	bn := b.N
	for i := p; i > 0; i-- {
		n := bn / i
		bn -= n
		go func() {
			defer wg.Done()
			for i := 0; i < n; i++ {
				fatalIfErr(b, store.PutTag(ctx, id, tag, getter))
			}
		}()
	}
	wg.Wait()
}

func BenchmarkPutRemoveTag(b *testing.B) {
	cids, getter := setup(b)
	db, err := leveldb.NewDatastore("", nil)
	fatalIfErr(b, err)
	store := NewTagCountedStore(db, nil)
	ctx := context.Background()
	id := cids[1]
	tag := datastore.NewKey("tag")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fatalIfErr(b, store.PutTag(ctx, id, tag, getter))
		fatalIfErr(b, store.RemoveTag(ctx, id, tag))
	}
}

func BenchmarkPutRemoveTag_P8(b *testing.B) {
	p := 8
	cids, getter := setup(b)
	db, err := leveldb.NewDatastore("", nil)
	fatalIfErr(b, err)
	store := NewTagCountedStore(db, nil)
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
				fatalIfErr(b, store.PutTag(ctx, id, tag, getter))
				fatalIfErr(b, store.RemoveTag(ctx, id, tag))
			}
		}()
	}
	wg.Wait()
}
