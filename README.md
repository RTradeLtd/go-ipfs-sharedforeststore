# Shared Forest Store

## Problem Statement

### The block datastore garbage collection problem

The default IPFS implementation uses pinning to keep track of what data to keep. It works similarity to how garbage collection works for heap allocation in memory managed programing languages. Superficially, this is a solved problem. But, just like how memory GC cycle slows down as heap grows, datastore GC also becomes unbearably slow for large IPFS deployments. (TODO: citation needed)

### Initial Solution: pinning by reference counting

@bonedaddy implemented a [much faster](https://medium.com/temporal-cloud/temporalx-vs-go-ipfs-official-node-benchmarks-8457037a77cf) alternative to pinning by counting the number of puts and deletes over a `Blockstore` interface.

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

#### Counting is for Speed

#### Choose Between Single Transaction or Progressive
