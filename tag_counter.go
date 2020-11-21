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

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
)

//TagCounted supports both TagStore and CounterStore interfaces.
//It is backed by a CounterStore and shares its counters.
type TagCounted struct {
	Counted
}

var _ TagCounterStore = (*TagCounted)(nil)

//NewTagCountedStore creates a new TagCounted from a transactional datastore.
func NewTagCountedStore(db datastore.TxnDatastore, opt *DatabaseOptions) *TagCounted {
	cs := NewCountedStore(db, opt)
	return &TagCounted{
		Counted: *cs,
	}
}

//txPutTag returns true if a new tag was added
func txPutTag(tx datastore.Txn, id cid.Cid, tag datastore.Key) (bool, error) {
	idtag := getTagKey(id, tag)
	if _, err := tx.Get(idtag); err != datastore.ErrNotFound {
		//tag already added, or some other error occurred
		return false, err //err is nil if tag is already on id
	}
	if err := tx.Put(idtag, nil); err != nil {
		return false, err
	}
	if err := tx.Put(idtag.Reverse(), nil); err != nil {
		return false, err
	}
	return true, nil
}

func (c *TagCounted) PutTag(ctx context.Context, id cid.Cid, tag datastore.Key, bg BlockGetter) error {
	return c.txWarp(ctx, func(tx *Tx) error {
		put, err := txPutTag(tx.transaction, id, tag)
		if !put {
			return err //err is nil if tag is already on id
		}
		_, err = tx.increment(id, bg, c.opt.LinkDecoder)
		return err
	})
}

func (c *TagCounted) HasTag(ctx context.Context, id cid.Cid, tag datastore.Key) (bool, error) {
	return c.ds.Has(getTagKey(id, tag))
}

func (c *TagCounted) GetTags(ctx context.Context, id cid.Cid) ([]datastore.Key, error) {
	prefix := newKeyFromCid(id, tagSuffixKey)
	rs, err := c.ds.Query(query.Query{
		Filters:  []query.Filter{query.FilterKeyPrefix{Prefix: prefix.String()}},
		KeysOnly: true,
	})
	if err != nil {
		return nil, err
	}
	es, err := rs.Rest()
	if err != nil {
		return nil, err
	}
	ps := len(prefix.String())
	tags := make([]datastore.Key, len(es))
	for i, e := range es {
		tags[i] = datastore.RawKey(e.Key[ps:])
	}
	return tags, nil
}

func (c *TagCounted) txRemoveTag(tx *Tx, id cid.Cid, tag datastore.Key) error {
	tk := getTagKey(id, tag)
	has, err := tx.transaction.Has(tk)
	if err != nil {
		return err
	}
	if !has {
		return nil
	}
	if err = tx.transaction.Delete(tk); err != nil {
		return err
	}
	if err = tx.transaction.Delete(tk.Reverse()); err != nil {
		return err
	}
	_, err = tx.decrement(id, c.opt.LinkDecoder)
	return err
}

func (c *TagCounted) RemoveTag(ctx context.Context, id cid.Cid, tag datastore.Key) error {
	return c.txWarp(ctx, func(tx *Tx) error {
		return c.txRemoveTag(tx, id, tag)
	})
}

//ReplaceTag adds the tag to all cids in update, and removes the tag from all other cids in the store.
func (c *TagCounted) ReplaceTag(ctx context.Context, update []cid.Cid, tag datastore.Key, bg BlockGetter) error {
	revPrefix := getTagKeyReversPrefix(tag)
	q := query.Query{
		Filters:  []query.Filter{query.FilterKeyPrefix{Prefix: revPrefix.String()}},
		KeysOnly: true,
	}
	return c.txWarp(ctx, func(tx *Tx) error {
		rs, err := c.ds.Query(q)
		if err != nil {
			return err
		}
		defer rs.Close()
		before := make(map[datastore.Key]struct{})
		es, err := rs.Rest()
		if err != nil {
			return err
		}

		// fill all existing keys with this tag to before map
		for _, e := range es {
			before[datastore.RawKey(e.Key).Reverse()] = struct{}{}
		}

		// for each cid in update, delete form before map or put new tag
		for _, uid := range update {
			ukey := getTagKey(uid, tag)
			if _, exist := before[ukey]; exist {
				delete(before, ukey)
				continue
			}
			put, err := txPutTag(tx.transaction, uid, tag)
			if err != nil {
				return err
			}
			if put {
				if _, err := tx.increment(uid, bg, c.opt.LinkDecoder); err != nil {
					return err
				}
			}
		}

		// for anything left in update, remove them
		for tk := range before {
			id, err := tagKeyToCid(tk)
			if err != nil {
				continue
			}
			if err := c.txRemoveTag(tx, id, tag); err != nil {
				return err
			}
		}
		return nil
	})
}
