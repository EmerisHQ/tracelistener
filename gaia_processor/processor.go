package gaia_processor

import (
	"fmt"

	"github.com/allinbits/tracelistener"
	"github.com/allinbits/tracelistener/config"
	"github.com/cosmos/cosmos-sdk/codec"
	gaia "github.com/cosmos/gaia/v4/app"
	"go.uber.org/zap"
)

type moduleProcessor interface {
	FlushCache() []tracelistener.WritebackOp
	OwnsKey(key []byte) bool
	Process(data tracelistener.TraceOperation) error
	ModuleName() string
	TableSchema() string
}

var p processor

type processor struct {
	l                *zap.SugaredLogger
	writeChan        chan tracelistener.TraceOperation
	writebackChan    chan []tracelistener.WritebackOp
	cdc              codec.Marshaler
	lastHeight       uint64
	chainName        string
	moduleProcessors []moduleProcessor
}

func New(logger *zap.SugaredLogger, cfg *config.Config) (tracelistener.DataProcessorInfos, error) {
	c := cfg.Gaia

	if c.ProcessorsEnabled == nil {
		c.ProcessorsEnabled = []string{"bank"}
	}

	var mp []moduleProcessor
	var tableSchemas []string

	for _, ep := range c.ProcessorsEnabled {
		p, err := processorByName(ep, logger)
		if err != nil {
			return tracelistener.DataProcessorInfos{}, err
		}

		mp = append(mp, p)
		tableSchemas = append(tableSchemas, p.TableSchema())
	}

	logger.Debugw("gaia processor initialized", "processors", c.ProcessorsEnabled)

	p = processor{
		chainName:        cfg.ChainName,
		l:                logger,
		writeChan:        make(chan tracelistener.TraceOperation),
		writebackChan:    make(chan []tracelistener.WritebackOp),
		moduleProcessors: mp,
	}

	cdc, _ := gaia.MakeCodecs()
	p.cdc = cdc

	go p.lifecycle()

	return tracelistener.DataProcessorInfos{
		OpsChan:            p.writeChan,
		WritebackChan:      p.writebackChan,
		DatabaseMigrations: tableSchemas,
	}, nil
}

func processorByName(name string, logger *zap.SugaredLogger) (moduleProcessor, error) {
	switch name {
	default:
		return nil, fmt.Errorf("unkonwn processor %s", name)
	case (&bankProcessor{}).ModuleName():
		return &bankProcessor{heightCache: map[bankCacheEntry]balanceWritebackPacket{}, l: logger}, nil
	case (&ibcConnectionsProcessor{}).ModuleName():
		return &ibcConnectionsProcessor{connectionsCache: map[connectionCacheEntry]connectionWritebackPacket{}, l: logger}, nil
	case (&liquidityPoolProcessor{}).ModuleName():
		return &liquidityPoolProcessor{poolsCache: map[uint64]poolWritebackPacket{}, l: logger}, nil
	case (&liquiditySwapsProcessor{}).ModuleName():
		return &liquiditySwapsProcessor{swapsCache: map[uint64]swapWritebackPacket{}, l: logger}, nil
	case (&delegationsProcessor{}).ModuleName():
		return &delegationsProcessor{heightCache: map[delegationCacheEntry]delegationWritebackPacket{}, l: logger}, nil
	}
}

func (p *processor) lifecycle() {
	for data := range p.writeChan {
		if data.BlockHeight != p.lastHeight && data.BlockHeight != 0 {
			wb := make([]tracelistener.WritebackOp, 0, len(p.moduleProcessors))

			for _, mp := range p.moduleProcessors {
				cd := mp.FlushCache()
				for _, entry := range cd {
					if entry.Data == nil {
						continue
					}

					for i := 0; i < len(entry.Data); i++ {
						entry.Data[i] = entry.Data[i].WithChainName(p.chainName)
					}
					wb = append(wb, entry)
				}
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
