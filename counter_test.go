// Copyright 2020 RTrade Technologies Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sharedforeststore

import (
	"context"
	"io"
	"testing"

	"github.com/ipfs/go-cid"
	leveldb "github.com/ipfs/go-ds-leveldb"
)

func TestCounter(t *testing.T) {
	t.Parallel()

	type testCase struct {
		node   int
		counts []int64
	}
	cases := []testCase{
		{node: 0, counts: []int64{1, 0, 0, 1, 0, 1}},
		{node: 1, counts: []int64{1, 1, 0, 2, 1, 3}},
		{node: 2, counts: []int64{1, 1, 1, 2, 2, 3}},
		{node: 3, counts: []int64{1, 1, 1, 3, 2, 3}},
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
		checkCounts(t, ctx, c.counts, cids, store)
	}

	checkFullStoreByIterator(t, ctx, cids, store)

	for _, c := range cases {
		count, err := store.Decrement(ctx, cids[c.node])
		fatalIfErr(t, err)
		if count != 0 {
			t.Fatalf("count should be 0 after decrement, but got %v", count)
		}
	}
}

func checkCounts(t testing.TB, ctx context.Context, exp []int64, cids []cid.Cid, store CounterStore) {
	for i, expCount := range exp {
		gotCount, err := store.GetCount(ctx, cids[i])
		fatalIfErr(t, err)
		if expCount != gotCount {
			t.Errorf("got count %v for index %v expected %v", gotCount, i, expCount)
		}
	}
}

func checkFullStoreByIterator(t testing.TB, ctx context.Context, cids []cid.Cid, store CounterStore) {
	it := store.KeysIterator("")
	expects := cid.NewSet()
	for _, id := range cids {
		expects.Add(id)
	}
	for {
		id, err := it.NextCid()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if !expects.Has(id) {
			t.Fatalf("unexpected cid %v found", id)
		}
		expects.Remove(id)
	}
	if expects.Len() != 0 {
		t.Fatalf("missed cids from iteration: %v", expects.Keys())
	}
}
