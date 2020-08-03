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
	"fmt"
	"sync"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	leveldb "github.com/ipfs/go-ds-leveldb"
)

type tagTestCase struct {
	node   int
	tag    string
	tags   []string
	counts []int64
}

var tagCases = []tagTestCase{
	{node: 0, tag: "A", tags: []string{"A"}, counts: []int64{1, 0, 0, 1, 0, 1}},
	{node: 1, tag: "A", tags: []string{"A"}, counts: []int64{1, 1, 0, 2, 1, 3}},
	{node: 2, tag: "B", tags: []string{"B"}, counts: []int64{1, 1, 1, 2, 2, 3}},
	{node: 3, tag: "C", tags: []string{"C"}, counts: []int64{1, 1, 1, 3, 2, 3}},
	{node: 0, tag: "A", tags: []string{"A"}, counts: []int64{1, 1, 1, 3, 2, 3}},
	{node: 0, tag: "B", tags: []string{"A", "B"}, counts: []int64{2, 1, 1, 3, 2, 3}},
}

func TestTagCounter(t *testing.T) {
	t.Parallel()

	cids, getter := setup(t)
	db, err := leveldb.NewDatastore("", nil)
	fatalIfErr(t, err)
	defer db.Close()
	store := NewTagCountedStore(db, nil)
	ctx := context.Background()

	for _, c := range tagCases {
		fatalIfErr(t, store.PutTag(ctx, cids[c.node], datastore.NewKey(c.tag), getter))
		checkCounts(t, ctx, c.counts, cids, store)
		checkTags(t, ctx, cids[c.node], c.tags, store)
	}
	checkFullStoreByIterator(t, ctx, cids, store)

	for _, c := range tagCases {
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
	defer db.Close()
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
	defer db.Close()
	store := NewTagCountedStore(db, nil)
	ctx := context.Background()
	tag := datastore.NewKey("tag")
	id := cids[1]
	wg := sync.WaitGroup{}
	wg.Add(p)
	b.ResetTimer()
	//start p go-routines
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
	defer db.Close()
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
	defer db.Close()
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

func checkTags(t testing.TB, ctx context.Context, id cid.Cid, tags []string, store TagStore) {
	gotTags, err := store.GetTags(ctx, id)
	fatalIfErr(t, err)
	if len(gotTags) != len(tags) {
		t.Fatalf("unexpected number of tags: %v", gotTags)
	}
	for i, tag := range tags {
		if !datastore.NewKey(tag).Equal(gotTags[i]) {
			t.Fatalf("expected tag: %v, got %v", datastore.NewKey(tag), gotTags[i])
		}
	}
}
