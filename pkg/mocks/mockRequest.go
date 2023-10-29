package mocks

import (
	"github.com/numaproj/numaflow-go/pkg/sourcer"
	"time"
)

type ReadRequest struct {
	CountValue uint64
	Timeout    time.Duration
}

func (r ReadRequest) TimeOut() time.Duration {
	return r.Timeout
}

func (r ReadRequest) Count() uint64 {
	return r.CountValue
}

type TestAckRequest struct {
	OffsetsValue []sourcer.Offset
}

func (ar TestAckRequest) Offsets() []sourcer.Offset {
	return ar.OffsetsValue
}
