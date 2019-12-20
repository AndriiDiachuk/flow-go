// (c) 2019 Dapper Labs - ALL RIGHTS RESERVED

package operation

import (
	"github.com/dgraph-io/badger/v2"

	"github.com/dapperlabs/flow-go/crypto"
	"github.com/dapperlabs/flow-go/model/collection"
)

func InsertCollections(hash crypto.Hash, collections []*collection.GuaranteedCollection) func(*badger.Txn) error {
	return insert(makePrefix(codeCollections, hash), collections)
}

func RetrieveCollections(hash crypto.Hash, collections *[]*collection.GuaranteedCollection) func(*badger.Txn) error {
	return retrieve(makePrefix(codeCollections, hash), collections)
}
