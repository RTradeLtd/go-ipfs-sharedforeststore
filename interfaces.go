package sharedforeststore

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
)

//BlockGetter returns the raw data referred to by cid.
//There are two use cases for BlockGetter:
// as a shared interface of all stores, and
// as a callback handler to provide the raw data only when needed.
type BlockGetter interface {
	//GetBlock is equivalent to Blockstore.Get
	GetBlock(context.Context, cid.Cid) ([]byte, error)
}

type CidIterator interface {
	//NextCid returns the next cid or io.EOF if the end is reached
	NextCid() (cid.Cid, error)
	//Close releases the resources used by this iterator for early exist
	Close() error
}

//ReadStore is the base interface for CounterStore and TaggedStore
type ReadStore interface {
	BlockGetter
	//GetSize is equivalent to Blockstore.GetSize
	GetBlockSize(context.Context, cid.Cid) (int, error)
	//KeysIterator replaces Blockstore.AllKeysChan.
	KeysIterator(prefix string) CidIterator
}

//CounterStore is a recursively counted BlockStore.
//When incrementing a counter, if it increased from 0 to 1,
// then the raw data is saved from BlockGetter and increment is called
// recursively to all the linked blocks.
//   Given the following linked graph:
//       A -> C -> D
//       B -> C -> D
//   And the following increment operations:
//       A, B, C, A
//   The result count will be:
//       A:2, B:1, C:3, D:1
//   This can be interpreted as:
//       A is added twice.
//       B is added once.
//       C is added once, and required by two other blocks, A and B, for a total count of 3
//       D is only required by C.
//   As you can see, the order of increments does not matter.
//Decrement removes the result of one increment.
type CounterStore interface {
	ReadStore
	//Decrement is equivalent to Blockstore.DeleteBlock.
	//Decrement is recursive if count hits 0.
	//If count >= 0, the reference counter got decreased by one.
	//If count == 0, the block referred to by cid is deleted.
	//If count == -1, the block referred to by cid does not exits
	// or was already deleted.
	//If err != nil, the operation failed and count should be ignored.
	Decrement(context.Context, cid.Cid) (int64, error)
	//GetCount is equivalent to Blockstore.Has.
	//The count could have changed by the time the function returned.
	//So it should not be used for decision making in a concurrent use case.
	GetCount(context.Context, cid.Cid) (int64, error)
	//Increment is equivalent to Blockstore.Put with recursive increment
	// for linked contents. The recursion only happens when count is
	// increased from 0 to 1.
	//The BlockGetter is responsible for providing any missing blocks during recursion.
	//If the BlockGetter returned any errors, an error is returned and no
	// count is modified.
	Increment(context.Context, cid.Cid, BlockGetter) (int64, error)
}

//TaggedStore is an extension of CounterStore where the count is replaced by a set of tags.
//At the cost of increased metadata size, this allows each operation to be idempotent,
// and there for safe to user over an undependable network connection.
//The tag can also be used for debugging and easily finding out who pinned which file.
type TaggedStore interface {
	ReadStore
	//PutTag adds a tag to contents referenced by the given cid
	PutTag(context.Context, cid.Cid, datastore.Key, BlockGetter) error
	//HasBlockTagged is the tagged version of GetCount
	HasBlockTagged(context.Context, cid.Cid, datastore.Key) (bool, error)
	//GetTags list tags on the cid, for administrative and debugging.
	//This function should be hidden from public facing APIs to make tags secret.
	GetTags(context.Context, cid.Cid) ([]datastore.Key, error)
	//RemoveTag removes a tag set on the cid, the contents are also removed
	//if there are no tags left.
	RemoveTag(context.Context, cid.Cid, datastore.Key) error
}

//TaggedCounterStore combines the features of both TaggedStore and CounterStore.
type TaggedCounterStore interface {
	TaggedStore
	CounterStore
}

//LinkDecoderFunc is a function that decodes a raw data according to cid to
//return linked cids.
type LinkDecoderFunc func(cid.Cid, []byte) ([]cid.Cid, error)
