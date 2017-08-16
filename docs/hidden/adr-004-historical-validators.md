# Enhanced validator queries

* [Problem](#problem)
* [Current State](#current-state)
* [Proposed Change](#proposed-change)
* [Benefits](#benefits)

## Problem

Right now, we can query the present validator set, but there is no history.
If you were offline for a long time, there is no way to reconstruct past validators.  This is needed for the light client and we agreed needs enhancement of the API.

## Current State

This can be problematic even for highly connected clients, as if I do:

1. Query abci state with proof -> proof @ H
2. Query header H and commit -> verify this state matches the header
3. Query validator set at H -> verify commit matches validators, these are secure from my known validators

If there is a new block and new validator change between 1 and 3, then the validator set I get at H+1 does not match the hash in header H and I have no way to get those validators.

## Proposed Change

I propose to store all validator hashes over the entire state of the blockchain in a merkle tree.  The key is the height it first became the validator.  The value is the validator hash of the set.  This allows us a number of efficient proofs of the validators over time, and even rapidly extend trust to headers far back in time.

Assuming there is a change to the set every 15 minutes, that is around 100 times a day, or 35000 times a year.  If we only store the 20 bytes validator hash, that doesn't require much space, 1MB after a few years....

The validator set would be stored separately in diffs for space efficiency. We can store the genesis validator set and diffs at each height it changes, which should be much smaller than the entire set.  If only 1-2 validators are involved in the change, it would be ~50 bytes per change, no problem.

However, when someone wants to get the validator set at height 1000000, I may have to run through 1000 changes since genesis to calculate it.  To reduce that problem, I could store "checkpoints" of the entire state every, say 50 changes, to have to apply maximum 49 diffs, and only use ~2% of the space of storing all validator sets ever.

This would be something like a WAL (WALs) to replay the validator set, and one iavltree to store the validator queries.

### Proofs

We would also have to add support for "range proofs".  If the validator set changes at height 1000 and again at 5000, and I query for the hash at 2300, the answer is the validator hash at height 1000 (which is still valid), but how do I prove that?  I need a second proof that the next key after 1000 is 5000, which is possible by exploiting the structure of the proofs, and ensuring they are sequential.  These two proofs with an algorithm, would prove this validator hash was valid from 1000 to 4999.

## Benefits

The most obvious benefit of being able to query and prove the validator set at all times, is the ability to safely make a 3 part query on a running system, without fearing the validators change before you query them.

However, it also makes historical proofs very doable.  I could verify that given only that I trust this block at 100000, when I joined the system, I could prove rapidly who the validators were at genesis, and then certify the genesis block from this chain, to know it for certainty, without having to reverse the entire set of queries.

I could do this for any height in the past, which is great if I store/submit a proof of abci state long in the past, it would always be verifiable.

## API Format

Just add a flag to the query:

GET `/validators?height=H`

```
{
  "jsonrpc": "2.0",
  "id": "",
  "result": {
    "block_height": 7816,
    "validators": [
      {
        "address": "7A956FADD20D3A5B2375042B2959F8AB172A058F",
        "pub_key": {
          "type": "ed25519",
          "data": "7B90EA87E7DC0C7145C8C48C08992BE271C7234134343E8A8E8008E617DE7B30"
        },
        "voting_power": 10,
      }
    ]
  },
  "error": ""
}
```
