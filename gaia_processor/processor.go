package gaia_processor

import (
	"github.com/cosmos/cosmos-sdk/codec"
	gaia "github.com/cosmos/gaia/v4/app"
	"go.uber.org/zap"
)

type moduleProcessor interface {
	FlushCache() tracelistener.WritebackOp
	OwnsKey(key []byte) bool
	Process(data tracelistener.TraceOperation) error
	ModuleName() string
}

var p processor

type processor struct {
	l                *zap.SugaredLogger
	writeChan        chan tracelistener.TraceOperation
	writebackChan    chan []tracelistener.WritebackOp
	cdc              codec.Marshaler
	lastHeight       uint64
	moduleProcessors []moduleProcessor
}

func New(logger *zap.SugaredLogger) (tracelistener.DataProcessorInfos, error) {
	p = processor{
		l:             logger,
		writeChan:     make(chan tracelistener.TraceOperation),
		writebackChan: make(chan []tracelistener.WritebackOp),
		moduleProcessors: []moduleProcessor{
			&bankProcessor{heightCache: map[cacheEntry]balanceWritebackPacket{}},
		},
	}

	cdc, _ := gaia.MakeCodecs()
	p.cdc = cdc

	go p.lifecycle()

	return tracelistener.DataProcessorInfos{
		OpsChan:       p.writeChan,
		WritebackChan: p.writebackChan,
		DatabaseMigrations: []string{
			createBalancesTable,
		},
	}, nil
}

func (p *processor) lifecycle() {
	for data := range p.writeChan {
		if data.BlockHeight != p.lastHeight && data.BlockHeight != 0 {
			wb := make([]tracelistener.WritebackOp, 0, len(p.moduleProcessors))

			for _, mp := range p.moduleProcessors {
				wb = append(wb, mp.FlushCache())
			}

			p.writebackChan <- wb

			p.l.Infow("processed new block", "height", p.lastHeight)

			p.lastHeight = data.BlockHeight
		}

		for _, mp := range p.moduleProcessors {
			if !mp.OwnsKey(data.Key) {
				continue
			}

			if err := mp.Process(data); err != nil {
				p.l.Errorw(
					"error while processing data",
					"data", data,
					"moduleName", mp.ModuleName())
			}
		}
	}
}
