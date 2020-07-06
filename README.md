# Shared Forest Store

A high level content store for IPFS.

## Problem Statement

### The block datastore garbage collection problem

The default IPFS implementation uses pinning to keep track of what data to keep. It works similarity to how garbage collection works for heap allocation in memory managed programing languages. Superficially, this is a solved problem. But, just like how memory GC cycle slows down as heap grows, datastore GC also becomes unbearably slow for large IPFS deployments. (TODO: citation needed)

### Initial Solution: pinning by reference counting

[postables](https://github.com/bonedaddy) implemented a [much faster](https://medium.com/temporal-cloud/temporalx-vs-go-ipfs-official-node-benchmarks-8457037a77cf) alternative to pinning by counting the number of puts and deletes over a `Blockstore` interface.s

Although that solution largely alleviated the slowdown, it exposed a few problems with the approach:

- A counter needs granted once delivery, which can not be dependent up on in RPC calls.
- Blockstore is too low level to manage recursive references internally, this blocks some efficiency improvements and API simplification.

## Solution

Shared Forest Store offers a collection of high level, idempotent, content store that combines the features of [Pinner](https://github.com/ipfs/go-ipfs-pinner) and [Blockstore](https://github.com/RTradeLtd/go-ipfs-blockstore) to create a "pinned" block store. It is design to be convent to use in an network facing, multiuser service.

### Technical Design

This design is not yet fully implemented.

#### Transactional Data Store

Both pinning metadata and block store operations are grouped into a single transaction. This offers true concurrency without locking and the database will never be in an inconsistent state. Any failed transition commits are retried automatically until either success or context cancellation.

#### Tagging is for Sharing

Tagging can be considered a keyed counter store, where each add is associated with a unique key. This not only offers idempotent operations, but by protecting the keys, users can share a single duplicating data store. For example, users could prefix their tags with a hash of the user's private key. By keeping this hash private, users can not delete each other's contents without any additional server side content protection logic.

#### Counting is for Speed

Where the additional features offered by tagging is unnicecery, counting is both faster and easier to use.

The `CounterStore` interface is primarily designed to aid in the transition from our internal counter store to this code base.

Counting is also an optional implementation for the internal references of a `TagStore`. Counting uses less metadata to keep track of internal references, while tagging offers more debugging keepabilities and quick reverse look up of why each block is needed.

#### Choose Between Single Transaction or Progressive

This content store library offers two transactional options when adding contents.

- Single Transaction: where blocks of an IPLD graph are either all saved or none at all.
- Progressive: where partial uploads are saved.

Single transaction is better for locally availed content and cases where partial adds can't be managed.

Progressive upload allows splitting the commit of an add operation into committing of individual blocks and accousated metadata. The progress of an add can also be reported.
