// (c) 2019 Dapper Labs - ALL RIGHTS RESERVED

package badger

import (
	"bytes"

	"github.com/dgraph-io/badger/v2"
	"github.com/pkg/errors"

	"github.com/dapperlabs/flow-go/crypto"
	"github.com/dapperlabs/flow-go/model/collection"
	"github.com/dapperlabs/flow-go/model/flow"
	"github.com/dapperlabs/flow-go/storage/badger/operation"
)

type Mutator struct {
	state *State
}

func (m *Mutator) Bootstrap(genesis *flow.Block) error {
	return m.state.db.Update(func(tx *badger.Txn) error {

		// check that the new identities are valid
		err := checkIdentitiesValidity(tx, genesis.NewIdentities)
		if err != nil {
			return errors.Wrap(err, "could not check identities validity")
		}

		// initialize the boundary of the finalized state
		err = initializeFinalizedBoundary(tx, genesis)
		if err != nil {
			return errors.Wrap(err, "could not initialize finalized boundary")
		}

		// store the block contents in the database
		err = storeBlockContents(tx, genesis)
		if err != nil {
			return errors.Wrap(err, "could not insert block payload")
		}

		// apply the block changes to the finalized state
		err = applyBlockChanges(tx, genesis)
		if err != nil {
			return errors.Wrap(err, "could not insert block deltas")
		}

		return nil
	})
}

func (m *Mutator) Extend(block *flow.Block) error {
	return m.state.db.Update(func(tx *badger.Txn) error {

		// check that the new identities are valid
		err := checkIdentitiesValidity(tx, block.NewIdentities)
		if err != nil {
			return errors.Wrap(err, "could not check identities validity")
		}

		// check that the block is a valid extension of the protocol state
		err = checkBlockValidity(tx, block.Header)
		if err != nil {
			return errors.Wrap(err, "could not check block validity")
		}

		// store the block contents in the database
		err = storeBlockContents(tx, block)
		if err != nil {
			return errors.Wrap(err, "could not insert block payload")
		}

		return nil
	})
}

type step struct {
	hash   crypto.Hash
	header flow.Header
}

func (m *Mutator) Finalize(hash crypto.Hash) error {
	return m.state.db.Update(func(tx *badger.Txn) error {

		// retrieve the block to make sure we have it
		var header flow.Header
		err := operation.RetrieveHeader(hash, &header)(tx)
		if err != nil {
			return errors.Wrap(err, "could not retrieve block")
		}

		// retrieve the current finalized state boundary
		var boundary uint64
		err = operation.RetrieveBoundary(&boundary)(tx)
		if err != nil {
			return errors.Wrap(err, "could not retrieve boundary")
		}

		// retrieve the hash of the boundary
		var head crypto.Hash
		err = operation.RetrieveHash(boundary, &head)(tx)
		if err != nil {
			return errors.Wrap(err, "could not retrieve head")
		}

		// in order to validate the validity of all changes, we need to iterate
		// through the blocks that need to be finalized from oldest to youngest;
		// we thus start at the youngest remember all of the intermediary steps
		// while tracing back until we reach the finalized state
		steps := []step{{hash: hash, header: header}}
		for !header.Parent.Equal(head) {
			hash = header.Parent
			err = operation.RetrieveHeader(header.Parent, &header)(tx)
			if err != nil {
				return errors.Wrapf(err, "could not retrieve parent (%x)", header.Parent)
			}
			steps = append(steps, step{hash: hash, header: header})
		}

		// now we can step backwards in order to go from oldest to youngest; for
		// each header, we reconstruct the block and then apply the related
		// changes to the protocol state
		var identities flow.IdentityList
		var collections []*collection.GuaranteedCollection
		for i := len(steps) - 1; i >= 0; i-- {

			// get the identities
			s := steps[i]
			err = operation.RetrieveIdentities(s.hash, &identities)(tx)
			if err != nil {
				return errors.Wrapf(err, "could not retrieve identities (%x)", hash)
			}

			// get the guaranteed collections
			err = operation.RetrieveCollections(s.hash, &collections)(tx)
			if err != nil {
				return errors.Wrapf(err, "could not retrieve collections (%x)", err)
			}

			// reconstruct block
			block := flow.Block{
				Header:                header,
				NewIdentities:         identities,
				GuaranteedCollections: collections,
			}

			// insert the deltas
			err = applyBlockChanges(tx, &block)
			if err != nil {
				return errors.Wrapf(err, "could not insert block deltas (%x)", s.hash)
			}
		}

		return nil
	})
}

func checkIdentitiesValidity(tx *badger.Txn, identities []flow.Identity) error {

	// check that we don't have duplicate identity entries
	lookup := make(map[flow.Identifier]struct{})
	for _, id := range identities {
		_, ok := lookup[id.NodeID]
		if ok {
			return errors.Errorf("duplicate node identity (%x)", id.NodeID)
		}
		lookup[id.NodeID] = struct{}{}
	}

	// for each identity, check it has a non-zero stake
	for _, id := range identities {
		if id.Stake == 0 {
			return errors.Errorf("invalid zero stake (%x)", id.NodeID)
		}
	}

	// for each identity, check it doesn't have a role yet
	for _, id := range identities {

		// check for role
		var role flow.Role
		err := operation.RetrieveRole(id.NodeID, &role)(tx)
		if errors.Cause(err) == badger.ErrKeyNotFound {
			continue
		}
		if err == nil {
			return errors.Errorf("identity role already exists (%x: %s)", id.NodeID, role)
		}
		return errors.Wrapf(err, "could not check identity role (%x)", id.NodeID)
	}

	// for each identity, check it doesn't have an address yet
	for _, id := range identities {

		// check for address
		var address string
		err := operation.RetrieveAddress(id.NodeID, &address)(tx)
		if errors.Cause(err) == badger.ErrKeyNotFound {
			continue
		}
		if err == nil {
			return errors.Errorf("identity address already exists (%x: %s)", id.NodeID, address)
		}
		return errors.Wrapf(err, "could not check identity address (%x)", id.NodeID)
	}

	return nil
}

func checkBlockValidity(tx *badger.Txn, header flow.Header) error {

	// get the boundary number of the finalized state
	var boundary uint64
	err := operation.RetrieveBoundary(&boundary)(tx)
	if err != nil {
		return errors.Wrap(err, "could not get boundary")
	}

	// get the hash of the latest finalized block
	var head crypto.Hash
	err = operation.RetrieveHash(boundary, &head)(tx)
	if err != nil {
		return errors.Wrap(err, "could not retrieve hash")
	}

	// get the first parent of the introduced block to check the number
	var parent flow.Header
	err = operation.RetrieveHeader(header.Parent, &parent)(tx)
	if err != nil {
		return errors.Wrap(err, "could not retrieve header")
	}

	// if new block number has a lower number, we can't finalize it
	if header.Number <= parent.Number {
		return errors.Errorf("block needs higher nummber (%d <= %d)", header.Number, parent.Number)
	}

	// NOTE: in the default case, the first parent is the boundary, se we don't
	// load the first parent twice almost ever; even in cases where we do, we
	// badger has efficietn caching, so no reason to complicate the algorithm
	// here to try avoiding one extra header loading

	// trace back from new block until we find a block that has the latest
	// finalized block as its parent
	for !header.Parent.Equal(head) {

		// get the parent of current block
		err = operation.RetrieveHeader(header.Parent, &header)(tx)
		if err != nil {
			return errors.Wrapf(err, "could not get parent (%x)", header.Parent)
		}

		// if its number is below current boundary, the block does not connect
		// to the finalized protocol state and would break database consistency
		if header.Number < boundary {
			return errors.Errorf("block doesn't connect to finalized state")
		}

	}

	return nil
}

func initializeFinalizedBoundary(tx *badger.Txn, genesis *flow.Block) error {

	// the initial finalized boundary needs to be height zero
	if genesis.Number != 0 {
		return errors.Errorf("invalid initial finalized boundary (%d != 0)", genesis.Number)
	}

	// the parent must be zero hash
	if !bytes.Equal(genesis.Parent, crypto.ZeroHash) {
		return errors.New("genesis parent must be zero hash")
	}

	// genesis should have no collections
	if len(genesis.GuaranteedCollections) > 0 {
		return errors.New("genesis should not contain collections")
	}

	// insert the initial finalized state boundary
	err := operation.InsertBoundary(genesis.Number)(tx)
	if err != nil {
		return errors.Wrap(err, "could not insert boundary")
	}

	return nil
}

func storeBlockContents(tx *badger.Txn, block *flow.Block) error {

	// insert the header into the DB
	err := operation.InsertHeader(&block.Header)(tx)
	if err != nil {
		return errors.Wrap(err, "could not insert header")
	}

	// NOTE: we might to improve this to insert an index, and then insert each
	// entity separately; this would allow us to retrieve the entities one by
	// one, instead of only by block

	// insert the identities into the DB
	err = operation.InsertIdentities(block.Hash(), block.NewIdentities)(tx)
	if err != nil {
		return errors.Wrap(err, "could not insert identities")
	}

	// insert the guaranteed collections into the DB
	err = operation.InsertCollections(block.Hash(), block.GuaranteedCollections)(tx)
	if err != nil {
		return errors.Wrap(err, "could not insert collections")
	}

	return nil
}

func applyBlockChanges(tx *badger.Txn, block *flow.Block) error {

	// insert the height to hash mapping for finalized block
	err := operation.InsertHash(block.Number, block.Hash())(tx)
	if err != nil {
		return errors.Wrap(err, "could not insert hash")
	}

	// update the finalized boundary number
	err = operation.UpdateBoundary(block.Number)(tx)
	if err != nil {
		return errors.Wrap(err, "could not update boundary")
	}

	// insert the information for each new identity
	for _, id := range block.NewIdentities {

		// insert the role
		err := operation.InsertRole(id.NodeID, id.Role)(tx)
		if err != nil {
			return errors.Wrapf(err, "could not insert role (%x)", id.NodeID)
		}

		// insert the address
		err = operation.InsertAddress(id.NodeID, id.Address)(tx)
		if err != nil {
			return errors.Wrapf(err, "could not insert address (%x)", id.NodeID)
		}

		// insert the stake delta
		err = operation.InsertDelta(block.Number, id.Role, id.NodeID, int64(id.Stake))(tx)
		if err != nil {
			return errors.Wrapf(err, "could not insert delta (%x)", id.NodeID)
		}
	}

	return nil
}
