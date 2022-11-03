package programs

import (
	"fmt"
	"sync"
)

type invalidatableEntry[TVal any] struct {
	Entry TVal // Immutable after initialization.

	isInvalid bool // Guarded by BlockDerivedData' lock.
}

// BlockDerivedData is a simple fork-aware OCC database for "caching" given types
// of data for a particular block.
//
// Since data are derived from external source, the database need not be
// durable and can be recreated on the fly.
//
// Furthermore, because data are derived, transaction validation looks
// a bit unusual when compared with a textbook OCC implementation.  In
// particular, the transaction's invalidator represents "real" writes to the
// canonical source, whereas the transaction's readSet/writeSet entries
// represent "real" reads from the canonical source.
type BlockDerivedData[TKey comparable, TVal any] struct {
	lock  sync.RWMutex
	items map[TKey]*invalidatableEntry[TVal]

	latestCommitExecutionTime LogicalTime

	invalidators chainedDerivedDataInvalidators[TVal] // Guarded by lock.
}

type TransactionDerivedData[TKey comparable, TVal any] struct {
	block *BlockDerivedData[TKey, TVal]

	// The start time when the snapshot first becomes readable (i.e., the
	// "snapshotTime - 1"'s transaction committed the snapshot view)
	snapshotTime LogicalTime

	// The transaction (or script)'s execution start time (aka TxIndex).
	executionTime LogicalTime

	readSet  map[TKey]*invalidatableEntry[TVal]
	writeSet map[TKey]TVal

	// When isSnapshotReadTransaction is true, invalidators must be empty.
	isSnapshotReadTransaction bool
	invalidators              chainedDerivedDataInvalidators[TVal]
}

func newEmptyBlockDerivedData[TKey comparable, TVal any](latestCommit LogicalTime) *BlockDerivedData[TKey, TVal] {
	return &BlockDerivedData[TKey, TVal]{
		items:                     map[TKey]*invalidatableEntry[TVal]{},
		latestCommitExecutionTime: latestCommit,
		invalidators:              nil,
	}
}

func NewEmptyBlockDerivedData[TKey comparable, TVal any]() *BlockDerivedData[TKey, TVal] {
	return newEmptyBlockDerivedData[TKey, TVal](ParentBlockTime)
}

// This variant is needed by the chunk verifier, which does not start at the
// beginning of the block.
func NewEmptyBlockDerivedDataWithOffset[TKey comparable, TVal any](offset uint32) *BlockDerivedData[TKey, TVal] {
	return newEmptyBlockDerivedData[TKey, TVal](LogicalTime(offset) - 1)
}

func (block *BlockDerivedData[TKey, TVal]) NewChildBlockDerivedData() *BlockDerivedData[TKey, TVal] {
	block.lock.RLock()
	defer block.lock.RUnlock()

	items := make(
		map[TKey]*invalidatableEntry[TVal],
		len(block.items))

	for key, entry := range block.items {
		items[key] = &invalidatableEntry[TVal]{
			Entry:     entry.Entry,
			isInvalid: false,
		}
	}

	return &BlockDerivedData[TKey, TVal]{
		items:                     items,
		latestCommitExecutionTime: ParentBlockTime,
		invalidators:              nil,
	}
}

func (block *BlockDerivedData[TKey, TVal]) NextTxIndexForTestingOnly() uint32 {
	return uint32(block.LatestCommitExecutionTimeForTestingOnly()) + 1
}

func (block *BlockDerivedData[TKey, TVal]) LatestCommitExecutionTimeForTestingOnly() LogicalTime {
	block.lock.RLock()
	defer block.lock.RUnlock()

	return block.latestCommitExecutionTime
}

func (block *BlockDerivedData[TKey, TVal]) EntriesForTestingOnly() map[TKey]*invalidatableEntry[TVal] {
	block.lock.RLock()
	defer block.lock.RUnlock()

	entries := make(
		map[TKey]*invalidatableEntry[TVal],
		len(block.items))
	for key, entry := range block.items {
		entries[key] = entry
	}

	return entries
}

func (block *BlockDerivedData[TKey, TVal]) InvalidatorsForTestingOnly() chainedDerivedDataInvalidators[TVal] {
	block.lock.RLock()
	defer block.lock.RUnlock()

	return block.invalidators
}

func (block *BlockDerivedData[TKey, TVal]) GetForTestingOnly(
	key TKey,
) *TVal {
	entry := block.get(key)
	if entry != nil {
		return &entry.Entry
	}
	return nil
}

func (block *BlockDerivedData[TKey, TVal]) get(
	key TKey,
) *invalidatableEntry[TVal] {
	block.lock.RLock()
	defer block.lock.RUnlock()

	return block.items[key]
}

func (block *BlockDerivedData[TKey, TVal]) unsafeValidate(
	item *TransactionDerivedData[TKey, TVal],
) RetryableError {
	if item.isSnapshotReadTransaction &&
		item.invalidators.ShouldInvalidateItems() {

		return newNotRetryableError(
			"invalid TransactionDerivedData: snapshot read can't invalidate")
	}

	if block.latestCommitExecutionTime >= item.executionTime {
		return newNotRetryableError(
			"invalid TransactionDerivedData: non-increasing time (%v >= %v)",
			block.latestCommitExecutionTime,
			item.executionTime)
	}

	if block.latestCommitExecutionTime+1 < item.snapshotTime &&
		(!item.isSnapshotReadTransaction ||
			item.snapshotTime != EndOfBlockExecutionTime) {

		return newNotRetryableError(
			"invalid TransactionDerivedData: missing commit range [%v, %v)",
			block.latestCommitExecutionTime+1,
			item.snapshotTime)
	}

	for _, entry := range item.readSet {
		if entry.isInvalid {
			return newRetryableError(
				"invalid TransactionDerivedDatas. outdated read set")
		}
	}

	applicable := block.invalidators.ApplicableInvalidators(
		item.snapshotTime)
	if applicable.ShouldInvalidateItems() {
		for _, entry := range item.writeSet {
			if applicable.ShouldInvalidateEntry(entry) {
				return newRetryableError(
					"invalid TransactionDerivedDatas. outdated write set")
			}
		}
	}

	return nil
}

func (block *BlockDerivedData[TKey, TVal]) validate(
	item *TransactionDerivedData[TKey, TVal],
) RetryableError {
	block.lock.RLock()
	defer block.lock.RUnlock()

	return block.unsafeValidate(item)
}

func (block *BlockDerivedData[TKey, TVal]) commit(
	item *TransactionDerivedData[TKey, TVal],
) RetryableError {
	block.lock.Lock()
	defer block.lock.Unlock()

	// NOTE: Instead of throwing out all the write entries, we can commit
	// the valid write entries then return error.
	err := block.unsafeValidate(item)
	if err != nil {
		return err
	}

	for key, entry := range item.writeSet {
		_, ok := block.items[key]
		if ok {
			// A previous transaction already committed an equivalent TransactionDerivedData
			// entry.  Since both TransactionDerivedData entry are valid, just reuse the
			// existing one for future transactions.
			continue
		}

		block.items[key] = &invalidatableEntry[TVal]{
			Entry:     entry,
			isInvalid: false,
		}
	}

	if item.invalidators.ShouldInvalidateItems() {
		for key, entry := range block.items {
			if item.invalidators.ShouldInvalidateEntry(
				entry.Entry) {

				entry.isInvalid = true
				delete(block.items, key)
			}
		}

		block.invalidators = append(
			block.invalidators,
			item.invalidators...)
	}

	// NOTE: We cannot advance commit time when we encounter a snapshot read
	// (aka script) transaction since these transactions don't generate new
	// snapshots.  It is safe to commit the entries since snapshot read
	// transactions never invalidate entries.
	if !item.isSnapshotReadTransaction {
		block.latestCommitExecutionTime = item.executionTime
	}
	return nil
}

func (block *BlockDerivedData[TKey, TVal]) newTransactionDerivedData(
	upperBoundExecutionTime LogicalTime,
	snapshotTime LogicalTime,
	executionTime LogicalTime,
	isSnapshotReadTransaction bool,
) (
	*TransactionDerivedData[TKey, TVal],
	error,
) {
	if executionTime < 0 || executionTime > upperBoundExecutionTime {
		return nil, fmt.Errorf(
			"invalid TransactionDerivedDatas: execution time out of bound: %v",
			executionTime)
	}

	if snapshotTime > executionTime {
		return nil, fmt.Errorf(
			"invalid TransactionDerivedDatas: snapshot > execution: %v > %v",
			snapshotTime,
			executionTime)
	}

	return &TransactionDerivedData[TKey, TVal]{
		block:                     block,
		snapshotTime:              snapshotTime,
		executionTime:             executionTime,
		readSet:                   map[TKey]*invalidatableEntry[TVal]{},
		writeSet:                  map[TKey]TVal{},
		isSnapshotReadTransaction: isSnapshotReadTransaction,
	}, nil
}

func (block *BlockDerivedData[TKey, TVal]) NewSnapshotReadTransactionDerivedData(
	snapshotTime LogicalTime,
	executionTime LogicalTime,
) (
	*TransactionDerivedData[TKey, TVal],
	error,
) {
	return block.newTransactionDerivedData(
		LargestSnapshotReadTransactionExecutionTime,
		snapshotTime,
		executionTime,
		true)
}

func (block *BlockDerivedData[TKey, TVal]) NewTransactionDerivedData(
	snapshotTime LogicalTime,
	executionTime LogicalTime,
) (
	*TransactionDerivedData[TKey, TVal],
	error,
) {
	return block.newTransactionDerivedData(
		LargestNormalTransactionExecutionTime,
		snapshotTime,
		executionTime,
		false)
}

func (item *TransactionDerivedData[TKey, TVal]) Get(key TKey) *TVal {
	writeEntry, ok := item.writeSet[key]
	if ok {
		return &writeEntry
	}

	readEntry := item.readSet[key]
	if readEntry != nil {
		return &readEntry.Entry
	}

	readEntry = item.block.get(key)
	if readEntry != nil {
		item.readSet[key] = readEntry
		return &readEntry.Entry
	}

	return nil
}

func (item *TransactionDerivedData[TKey, TVal]) Set(key TKey, val TVal) {
	item.writeSet[key] = val
}

func (item *TransactionDerivedData[TKey, TVal]) AddInvalidator(
	invalidator DerivedDataInvalidator[TVal],
) {
	if invalidator == nil || !invalidator.ShouldInvalidateItems() {
		return
	}

	item.invalidators = append(
		item.invalidators,
		derivedDataInvalidatorAtTime[TVal]{
			DerivedDataInvalidator: invalidator,
			executionTime:          item.executionTime,
		})
}

func (item *TransactionDerivedData[TKey, TVal]) Validate() RetryableError {
	return item.block.validate(item)
}

func (item *TransactionDerivedData[TKey, TVal]) Commit() RetryableError {
	return item.block.commit(item)
}
