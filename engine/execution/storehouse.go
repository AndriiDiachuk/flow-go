package execution

import (
	"context"
	"hash"
	"time"

	"github.com/onflow/flow-go/fvm/storage/derived"
	"github.com/onflow/flow-go/fvm/storage/snapshot"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/module/executiondatasync/execution_data"
	"github.com/onflow/flow-go/module/mempool/entity"
)

type RegisterStore interface {
	// Depend on OnDiskRegisterStore.Init
	// Depend on InMemoryRegisterStore.InitWithLatestHeight
	Init() error

	GetRegister(height uint64, blockID flow.Identifier, register flow.RegisterID) (flow.RegisterValue, error)
	GetRegisters(height uint64, blockID flow.Identifier, registers []flow.RegisterID) ([]flow.RegisterValue, error)

	// SaveRegister will always save to InMemoryRegisterStore first
	// Depend on InMemoryRegisterStore.SaveRegister
	SaveRegister(height uint64, blockID flow.Identifier, registers []flow.RegisterEntry) error

	// Depend on ExecutedFinalizedWAL.Append
	// Depend on OnDiskRegisterStore.SaveRegister
	// Depend on FinalizedReader's GetFinalizedBlockIDAtHeight
	// OnBlockFinalized will trigger the check of whether a block at the next height becomes finalized and executed.
	// the next height is the existing finalized and executed block's height + 1.
	// If a block at next height becomes finalized and executed, save the registers of the block to
	// OnDiskRegisterStore, and then prune the height in InMemoryRegisterStore
	OnBlockFinalized() error

	FinalizedAndExecutedHeight() (uint64, error)

	Checkpoint() (height uint64, err error)
}

type FinalizedReader interface {
	GetFinalizedBlockIDAtHeight(height uint64) (flow.Identifier, error)
}

type InMemoryRegisterStore interface {
	// Init with last finalized and executed height, which will be set as pruned height
	InitWithLatestHeight(height uint64) error
	Prune(height uint64) error
	PrunedHeight() uint64

	GetRegister(height uint64, blockID flow.Identifier, register flow.RegisterID) (flow.RegisterValue, error)
	GetRegisters(height uint64, blockID flow.Identifier, registers []flow.RegisterID) ([]flow.RegisterValue, error)
	SaveRegister(height uint64, blockID flow.Identifier, registers []flow.RegisterEntry) error
}

type OnDiskRegisterStore interface {
	GetRegister(height uint64, register flow.RegisterID) (flow.RegisterValue, error)
	GetRegisters(height uint64) ([]flow.RegisterEntry, error)
	SaveRegister(height uint64, registers []flow.RegisterEntry) error
	// latest finalized and executed height
	Latest() (height uint64)
}

type IngestionEngine interface {
	Ready() error
	// Depend on ComputerManager's ComputeBlock
	// Depend on ExecutionState's SaveExecutionResults
	ExecuteBlock() error
}

type scriptExecutor interface {
	// ExecuteScriptAtBlockID executes a script at the given Block id
	// Depend on ReadyOnlyExecutionState.NewStorageSnapshot
	ExecuteScriptAtBlockID(
		ctx context.Context,
		script []byte,
		arguments [][]byte,
		blockID flow.Identifier,
	) ([]byte, error)

	GetRegisterAtBlockID(
		ctx context.Context,
		owner,
		key []byte,
		blockID flow.Identifier) ([]byte, error)
}

type ReadyOnlyExecutionState interface {
	// NewStorageSnapshot creates a new ready-only view at the given state commitment.
	// Return storehouse API
	// Depend on ledger.GetSingleValue(statecommitment, registerID) (depcreated)
	// Depend on Storehouse.GetRegister
	// TODO: add height
	NewStorageSnapshot(flow.StateCommitment) snapshot.StorageSnapshot

	// StateCommitmentByBlockID returns the final state commitment for the provided block ID.
	StateCommitmentByBlockID(context.Context, flow.Identifier) (flow.StateCommitment, error)

	// HasState returns true if the state with the given state commitment exists in memory
	HasState(flow.StateCommitment) bool

	GetExecutionResultID(context.Context, flow.Identifier) (flow.Identifier, error)
}

type ExecutionState interface {
	ReadyOnlyExecutionState

	// Depend on badger DB
	SaveExecutionResults(
		ctx context.Context,
		result *ComputationResult,
	) error
}

type ComputerManager interface {
	// Depend on Computer's ExecuteBlock
	ComputeBlock(
		ctx context.Context,
		parentBlockExecutionResultID flow.Identifier,
		block *entity.ExecutableBlock,
		snapshot snapshot.StorageSnapshot,
	) (
		*ComputationResult,
		error,
	)
}

type Computer interface {
	// Depend on ResultCollector
	ExecuteBlock(
		ctx context.Context,
		parentBlockExecutionResultID flow.Identifier,
		block *entity.ExecutableBlock,
		snapshot snapshot.StorageSnapshot,
		derivedBlockData *derived.DerivedBlockData,
	) (
		*ComputationResult,
		error,
	)
}

type collectionInfo struct {
	blockId    flow.Identifier
	blockIdStr string

	collectionIndex int
	*entity.CompleteCollection

	isSystemTransaction bool
}

type ResultCollector interface {
	// Depend on ViewCommiter
	CommitCollection(
		collection collectionInfo,
		startTime time.Time,
		collectionExecutionSnapshot *snapshot.ExecutionSnapshot,
	) error

	// Depend on ExecutionDataProvider.Provide
	Finalize(ctx context.Context) (*ComputationResult, error)
}

type ExecutionDataProvider interface {
	Provide(
		ctx context.Context,
		blockHeight uint64, // for pruning
		executionData *execution_data.BlockExecutionData,
	) (flow.Identifier, error)
}

type ViewCommiter interface {
	// Depend on ledger.Prove (proof)
	// Depend on ledger.Set to save the trie update (deprecated)
	// Depend on RegisterStore.Store
	CommitView(
		*snapshot.ExecutionSnapshot,
		flow.StateCommitment,
	) (
		flow.StateCommitment,
		[]byte, // proof
		*ledger.TrieUpdate,
		error,
	)
}

type ExecutedFinalizedWAL interface {
	Append(height uint64, trieUpdates []*ledger.TrieUpdate) error

	// GetLatest returns the latest height in the WAL.
	Latest() (uint64, error)

	GetReader(height uint64) (WALReader, error)
}

type WALReader interface {
	// Next returns the next height and trie updates in the WAL.
	// It returns EOF when there are no more entries.
	Next() (height uint64, trieUpdates []*ledger.TrieUpdate, err error)
}

type Ledger interface {
	// Depend on RegisterlessTrieCheckpointReader.ReadChecpoint
	Init() error
	// Depend on RegisterlessTrie
	Prove(height uint64, query *ledger.Query) (proof ledger.Proof, err error)
	// baseState is the state at the previous height (height - 1)
	Update(height uint64, baseState ledger.State, updates flow.RegisterEntries) (newState ledger.State, trieUpdate *ledger.TrieUpdate, err error)
	// Prune with finalized and executed height
	Prune(height uint64) error
	PrunedHeight() uint64
}

type TrieNode struct {
	leftChild      *TrieNode
	rightChild     *TrieNode
	height         int
	leafNodePath   ledger.Path
	leafNodeHash   hash.Hash
	cachedNodeHash hash.Hash
}

type RegisterlessTrie interface {
	IsEmpty() bool
	RootNode() *TrieNode
	RootHash() ledger.RootHash
	AllocatedRegCount() uint64
	AllocatedRegSize() uint64
	String() string
	UnsafeProofs(paths []ledger.Path) *ledger.TrieBatchProof
	UnsafeValueSizes(paths []ledger.Path) []int
	Extend(updates flow.RegisterEntries) (RegisterlessTrie, error)
}

type RegisterlessTrieCheckpointWriter interface {
	// Store the latest finalized and executed block into a checkpoint
	StoreCheckpoint(height uint64, trie RegisterlessTrie) error
}

type RegisterlessTrieCheckpointReader interface {
	// Checkpoint contains only a single Trie for
	// the latest finalized and executed block
	ReadChecpoint() (height uint64, trie RegisterlessTrie, err error)
}