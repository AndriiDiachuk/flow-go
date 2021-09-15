package signature

import (
	"fmt"
	"sync"

	"github.com/onflow/flow-go/crypto"
	"github.com/onflow/flow-go/engine"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/module/signature"
)

// WeightedSignatureAggregator implements consensus/hotstuff.WeightedSignatureAggregator
type WeightedSignatureAggregator struct {
	*signature.SignatureAggregatorSameMessage                              // low level crypto aggregator, agnostic of weights and flow IDs
	signers                                   []flow.Identity              // all possible signers, defining a canonical order
	idToIndex                                 map[flow.Identifier]int      // map node identifiers to indices
	idToWeights                               map[flow.Identifier]uint64   // weight of each signer
	totalWeight                               uint64                       // weight collected
	lock                                      sync.RWMutex                 // lock for atomic updates
	collectedIDs                              map[flow.Identifier]struct{} // map of collected IDs
}

// NewWeightedSignatureAggregator returns a weighted aggregator initialized with the input data.
//
// A weighted aggregator is used for one aggregation only. A new instance should be used for each use.
func NewWeightedSignatureAggregator(
	signers []flow.Identity, // list of all possible signers
	message []byte, // message to get an aggregated signature for
	dsTag string, // domain separation tag used by the signature
) (*WeightedSignatureAggregator, error) {

	// build a low level crypto aggregator
	publicKeys := make([]crypto.PublicKey, 0, len(signers))
	for _, id := range signers {
		publicKeys = append(publicKeys, id.StakingPubKey)
	}
	agg, err := signature.NewSignatureAggregatorSameMessage(message, dsTag, publicKeys)
	if err != nil {
		return nil, fmt.Errorf("new signature aggregator failed: %w", err)
	}

	// build the weighted aggregator
	weightedAgg := &WeightedSignatureAggregator{
		SignatureAggregatorSameMessage: agg,
		signers:                        signers,
	}

	// build the internal maps for a faster look-up
	for i, id := range signers {
		weightedAgg.idToIndex[id.NodeID] = i
		weightedAgg.idToWeights[id.NodeID] = id.Stake
	}
	return weightedAgg, nil
}

// Verify verifies the signature under the stored public and message.
//
// The function errors:
//  - engine.InvalidInputError if signerID is invalid (not a consensus participant)
//  - module/signature.ErrInvalidFormat if signerID is valid but signature is cryptographically invalid
//  - random error if the execution failed
// The function is not thread-safe.
func (s *WeightedSignatureAggregator) Verify(signerID flow.Identifier, sig crypto.Signature) error {
	index, ok := s.idToIndex[signerID]
	if !ok {
		return engine.NewInvalidInputErrorf("couldn't find signerID %s in the index map", signerID)
	}
	ok, err := s.SignatureAggregatorSameMessage.Verify(index, sig)
	if err != nil {
		return fmt.Errorf("couldn't verify signature from %s: %w", signerID, err)
	}
	if !ok {
		return signature.ErrInvalidFormat
	}
	return nil
}

// TrustedAdd adds a signature to the internal set of signatures and adds the signer's
// weight to the total collected weight, iff the signature is _not_ a duplicate.
//
// The total weight of all collected signatures (excluding duplicates) is returned regardless
// of any returned error.
// The function errors
//  - engine.InvalidInputError if signerID is invalid (not a consensus participant)
//  - engine.DuplicatedEntryError if the signer has been already added
// The function is thread-safe.
func (s *WeightedSignatureAggregator) TrustedAdd(signerID flow.Identifier, sig crypto.Signature) (uint64, error) {
	// get the total weight safely
	collectedWeight := s.TotalWeight()

	// get the index
	index, ok := s.idToIndex[signerID]
	if !ok {
		return collectedWeight, engine.NewInvalidInputErrorf("couldn't find signerID %s in the index map", signerID)
	}
	// get the weight
	weight, ok := s.idToWeights[signerID]
	if !ok {
		return collectedWeight, engine.NewInvalidInputErrorf("couldn't find signerID %s in the weight map", signerID)
	}

	// atomically update the signatures pool and the total weight
	s.lock.Lock()
	defer s.lock.Unlock()

	// check for double-voters.
	_, ok = s.collectedIDs[signerID]
	if ok {
		return collectedWeight, engine.NewDuplicatedEntryErrorf("SigneID %s was already added", signerID)
	}

	err := s.SignatureAggregatorSameMessage.TrustedAdd(index, sig)
	if err != nil {
		return collectedWeight, fmt.Errorf("Trusted add has failed: %w", err)
	}

	s.collectedIDs[signerID] = struct{}{}
	collectedWeight += weight
	s.totalWeight = collectedWeight
	return collectedWeight, nil
}

// TotalWeight returns the total weight presented by the collected signatures.
//
// The function is thread-safe
func (s *WeightedSignatureAggregator) TotalWeight() uint64 {
	s.lock.RLock()
	defer s.lock.RUnlock()
	collectedWeight := s.totalWeight
	return collectedWeight
}

// Aggregate aggregates the signatures and returns the aggregated signature.
//
// The function is thread-safe.
// Aggregate attempts to aggregate the internal signatures and returns the resulting signature data.
// The function performs a final verification and errors if the aggregated signature is not valid. This is
// required for the function safety since "TrustedAdd" allows adding invalid signatures.
//
// TODO : When compacting the list of signers, update the return from []flow.Identifier
// to a compact bit vector.
func (s *WeightedSignatureAggregator) Aggregate() ([]flow.Identifier, []byte, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Aggregate includes the safety check of the aggregated signature
	indices, aggSignature, err := s.SignatureAggregatorSameMessage.Aggregate()
	if err != nil {
		return nil, nil, fmt.Errorf("Aggregate has failed: %w", err)
	}
	signerIDs := make([]flow.Identifier, 0, len(indices))
	for _, i := range indices {
		signerIDs = append(signerIDs, s.signers[i].NodeID)
	}

	return signerIDs, aggSignature, nil
}
