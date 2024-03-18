package backend

import (
	"context"
	"time"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/ccf"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/onflow/flow-go/engine"
	"github.com/onflow/flow-go/engine/access/state_stream"
	"github.com/onflow/flow-go/engine/access/subscription"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/module/counters"
)

type AccountStatusesResponse struct {
	BlockID       flow.Identifier
	Height        uint64
	AccountEvents map[string]flow.EventsList
	MessageIndex  uint64
}

// AccountStatusesBackend is a struct representing a backend implementation for subscribing to account statuses changes.
type AccountStatusesBackend struct {
	log            zerolog.Logger
	broadcaster    *engine.Broadcaster
	sendTimeout    time.Duration
	responseLimit  float64
	sendBufferSize int

	getStartHeight  subscription.GetStartHeightFunc
	eventsRetriever EventsRetriever
}

// subscribeAccountStatuses subscribes to account status changes starting from a specific block ID
// and block height, with an optional status filter.
// Errors:
// - codes.InvalidArgument: If start height before root height, or both startBlockID and startHeight are provided.
// - codes.ErrNotFound`: For unindexed start blockID or for unindexed start height.
// - codes.Internal: If there is an internal error.
func (b *AccountStatusesBackend) subscribeAccountStatuses(
	ctx context.Context,
	startBlockID flow.Identifier,
	startHeight uint64,
	filter state_stream.EventFilter,
) subscription.Subscription {
	nextHeight, err := b.getStartHeight(ctx, startBlockID, startHeight)
	if err != nil {
		return subscription.NewFailedSubscription(err, "could not get start height")
	}

	messageIndex := counters.NewMonotonousCounter(0)
	sub := subscription.NewHeightBasedSubscription(b.sendBufferSize, nextHeight, b.getAccountStatusResponseFactory(&messageIndex, filter))
	go subscription.NewStreamer(b.log, b.broadcaster, b.sendTimeout, b.responseLimit, sub).Stream(ctx)

	return sub
}

// SubscribeAccountStatusesFromStartBlockID subscribes to the streaming of account status changes starting from
// a specific block ID with an optional status filter.
func (b *AccountStatusesBackend) SubscribeAccountStatusesFromStartBlockID(
	ctx context.Context,
	startBlockID flow.Identifier,
	filter state_stream.EventFilter,
) subscription.Subscription {
	return b.subscribeAccountStatuses(ctx, startBlockID, 0, filter)
}

// SubscribeAccountStatusesFromStartHeight subscribes to the streaming of account status changes starting from
// a specific block height, with an optional status filter.
func (b *AccountStatusesBackend) SubscribeAccountStatusesFromStartHeight(
	ctx context.Context,
	startHeight uint64,
	filter state_stream.EventFilter,
) subscription.Subscription {
	return b.subscribeAccountStatuses(ctx, flow.ZeroID, startHeight, filter)
}

// SubscribeAccountStatusesFromLatestBlock subscribes to the streaming of account status changes starting from a
// latest sealed block, with an optional status filter.
func (b *AccountStatusesBackend) SubscribeAccountStatusesFromLatestBlock(
	ctx context.Context,
	filter state_stream.EventFilter,
) subscription.Subscription {
	return b.subscribeAccountStatuses(ctx, flow.ZeroID, 0, filter)
}

// getAccountStatusResponseFactory returns a function that returns the account statuses response for a given height.
func (b *AccountStatusesBackend) getAccountStatusResponseFactory(messageIndex *counters.StrictMonotonousCounter, filter state_stream.EventFilter) subscription.GetDataByHeightFunc {
	return func(ctx context.Context, height uint64) (interface{}, error) {
		var err error
		eventsResponse, err := b.eventsRetriever.GetAllEventsResponse(ctx, height)
		if err != nil {
			return nil, err
		}
		filteredProtocolEvents := filter.Filter(eventsResponse.Events)
		allAccountProtocolEvents := map[string]flow.EventsList{}

		for _, event := range filteredProtocolEvents {
			data, err := ccf.Decode(nil, event.Payload)
			if err != nil {
				b.log.Info().Err(err).Msg("could not decode event payload")
				continue
			}

			cdcEvent, ok := data.(cadence.Event)
			if !ok {
				b.log.Error().Err(err).Msg("could not cast to cadence event")
				continue
			}

			fieldValues := cdcEvent.GetFieldValues()
			fields := cdcEvent.GetFields()
			if fieldValues == nil || fields == nil {
				continue
			}

			accountField := state_stream.GetCoreEventAccountFieldFilter(event.Type)
			for i, field := range fields {
				if field.Identifier == accountField.FieldName {
					fieldValue := fieldValues[i].String()
					allAccountProtocolEvents[fieldValue] = append(allAccountProtocolEvents[fieldValue], event)
				}
			}
		}

		index := messageIndex.Value()
		response := &AccountStatusesResponse{
			BlockID:       eventsResponse.BlockID,
			Height:        eventsResponse.Height,
			AccountEvents: allAccountProtocolEvents,
			MessageIndex:  index,
		}

		if ok := messageIndex.Set(index + 1); !ok {
			return nil, status.Errorf(codes.Internal, "message index already incremented to %d", messageIndex.Value())
		}

		return response, nil
	}
}
