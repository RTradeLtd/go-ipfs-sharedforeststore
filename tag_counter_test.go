package sharedforeststore

import (
	"context"
	"testing"

	"github.com/ipfs/go-datastore"
	leveldb "github.com/ipfs/go-ds-leveldb"
)

func TestTafCounter(t *testing.T) {
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
