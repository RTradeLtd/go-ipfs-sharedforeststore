package sharedforeststore

import (
	"context"
	"io"
	"testing"

	leveldb "github.com/ipfs/go-ds-leveldb"
)

func TestCounter(t *testing.T) {
	type testCase struct {
		node   int
		counts []int64
	}

	cases := []testCase{
		testCase{node: 0, counts: []int64{1, 0, 0, 1, 0, 1}},
		testCase{node: 1, counts: []int64{1, 1, 0, 2, 1, 3}},
		testCase{node: 2, counts: []int64{1, 1, 1, 2, 2, 3}},
		testCase{node: 3, counts: []int64{1, 1, 1, 3, 2, 3}},
	}

	cids, getter := setup(t)
	db, err := leveldb.NewDatastore("", nil)
	fatalIfErr(t, err)
	store := NewCountedStore(db, nil)
	ctx := context.Background()

	for _, c := range cases {
		count, err := store.Increment(ctx, cids[c.node], getter)
		fatalIfErr(t, err)
		if count != c.counts[c.node] {
			t.Fatalf("count %v != %v", count, c.counts[c.node])
		}

		for i, expCount := range c.counts {
			gotCount, err := store.GetCount(ctx, cids[i])
			fatalIfErr(t, err)
			if expCount != gotCount {
				t.Errorf("case %v got count %v for index %v expected %v", c, gotCount, i, expCount)
			}
		}
	}

	it := store.KeysIterator("")
	storeSize := 0
	for {
		_, err := it.NextCid()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		storeSize++
	}
	if storeSize != len(cids) {
		t.Fatalf("KeysIterator expected %v items, but got %v items", len(cids), storeSize)
	}

	for _, c := range cases {
		count, err := store.Decrement(ctx, cids[c.node])
		fatalIfErr(t, err)
		if count != 0 {
			t.Fatalf("count should be 0 after decrement, but got %v", count)
		}
	}
}
