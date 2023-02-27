package cache

import (
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/module"
)

type OnEntityEjected func(ejectedEntity flow.Entity)

// HeroCacheDistributor wraps module.HeroCacheMetrics and allows subscribers to receive events
// for ejected entries from cache.
type HeroCacheDistributor struct {
	module.HeroCacheMetrics
	consumers []OnEntityEjected
}

var _ module.HeroCacheMetrics = (*HeroCacheDistributor)(nil)

func NewDistributor(heroCacheMetrics module.HeroCacheMetrics) *HeroCacheDistributor {
	return &HeroCacheDistributor{
		HeroCacheMetrics: heroCacheMetrics,
	}
}

func (d *HeroCacheDistributor) AddConsumer(consumer OnEntityEjected) {
	d.consumers = append(d.consumers, consumer)
}

func (d *HeroCacheDistributor) OnEntityEjectionDueToFullCapacity(ejectedEntity flow.Entity) {
	// report to parent metrics
	d.HeroCacheMetrics.OnEntityEjectionDueToFullCapacity(ejectedEntity)
	// report to extra consumers
	for _, consumer := range d.consumers {
		consumer(ejectedEntity)
	}
}
