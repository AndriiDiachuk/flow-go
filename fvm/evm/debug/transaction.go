package debug

import (
	"encoding/json"
	"fmt"

	"github.com/onflow/go-ethereum/eth/tracers"
)

type EVMTracer interface {
	TxTracer() tracers.Tracer
	Collect()
}

var _ EVMTracer = &CallTracer{}

type CallTracer struct {
	tracer   tracers.Tracer
	uploader Uploader
}

func NewEVMCallTracer(uploader Uploader) (*CallTracer, error) {
	tracerType := json.RawMessage(`{ "onlyTopCall": true }`)
	tracer, err := tracers.DefaultDirectory.New("callTracer", &tracers.Context{}, tracerType)
	if err != nil {
		return nil, err
	}

	return &CallTracer{
		tracer:   tracer,
		uploader: uploader,
	}, nil
}

func (t *CallTracer) TxTracer() tracers.Tracer {
	return t.tracer
}

func (t *CallTracer) Collect() {
	res, err := t.tracer.GetResult()
	if err != nil {
		fmt.Println(err)
	}

	err = t.uploader.Upload(res)
	if err != nil {
		fmt.Println(err)
	}
}

var _ EVMTracer = &NopTracer{}

type NopTracer struct{}

func (n NopTracer) TxTracer() tracers.Tracer {
	return nil
}

func (n NopTracer) Collect() {}
