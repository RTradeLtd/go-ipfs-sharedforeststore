package sharedforeststore

import (
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
)

type BlockGetter func(cid.Cid) ([]byte, error)

type CidIterator interface {
	ReadCids(max int) ([]cid.Cid, error)
}

type ReadStore interface {
	//GetCount is equivalent to Blockstore.Get
	GetBlock(cid.Cid) ([]byte, error)
	//GetSize is equivalent to Blockstore.GetSize
	GetSize(cid.Cid) (int, error)
	//KeysIterator replaces Blockstore.AllKeysChan.
	KeysIterator() CidIterator
}

type CounterStore interface {
	ReadStore
	//Decrement is equivalent to Blockstore.DeleteBlock.
	//Decrement is recursive if count hits 0.
	//If count >= 0, the reference counter got decreased by one.
	//If count == 0, the block referred to by cid is deleted.
	//If count == -1, the block referred to by cid does not exits
	// or was already deleted.
	//If err != nil, the operation failed and count should be ignored.
	Decrement(cid.Cid) (int64, error)
	//GetCount is equivalent to Blockstore.Has.
	//The count could have changed by the time the function returned.
	//So it should not be used for decision making in a concurrent use case.
	GetCount(cid.Cid) (int64, error)
	//Increment is equivalent to Blockstore.Put with recursive increment
	// for linked contents. The recursion only happens when count is
	// increased from 0 to 1.
	//The BlockGetter is responsible for providing any missing blocks during recursion.
	//If the BlockGetter returned any errors, an error is returned and no
	// count is modified.
	Increment(cid.Cid, BlockGetter) (int64, error)
}

type TaggedStore interface {
	ReadStore
	SetTag(datastore.Key, cid.Cid, BlockGetter) error
	GetTags(cid.Cid) ([]datastore.Key, error)
	RemoveTag(datastore.Key, cid.Cid) error
}
