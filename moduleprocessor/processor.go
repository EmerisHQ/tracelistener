package moduleprocessor

import (
	"fmt"

	"google.golang.org/grpc"

	tracelistener2 "github.com/allinbits/tracelistener"
	config2 "github.com/allinbits/tracelistener/config"

	models "github.com/allinbits/demeris-backend-models/tracelistener"

	"github.com/cosmos/cosmos-sdk/codec"
	"go.uber.org/zap"
)

type Module interface {
	FlushCache() []tracelistener2.WritebackOp
	OwnsKey(key []byte) bool
	Process(data tracelistener2.TraceOperation) error
	ModuleName() string
	TableSchema() string
}

var defaultProcessors = []string{
	//"auth",
	//"bank",
	//"delegations",
	"validators",
	//"unbonding_delegations",
	//"ibc_clients",
	//"ibc_channels",
	//"ibc_connections",
	//"ibc_denom_traces",
}

type Processor struct {
	l                *zap.SugaredLogger
	writeChan        chan tracelistener2.TraceOperation
	writebackChan    chan []tracelistener2.WritebackOp
	errorsChan       chan error
	cdc              codec.Marshaler
	migrations       []string
	lastHeight       uint64
	chainName        string
	moduleProcessors []Module
}

func (p *Processor) OpsChan() chan tracelistener2.TraceOperation {
	return p.writeChan
}

func (p *Processor) WritebackChan() chan []tracelistener2.WritebackOp {
	return p.writebackChan
}

func (p *Processor) DatabaseMigrations() []string {
	return p.migrations
}

func (p *Processor) ErrorsChan() chan error {
	return p.errorsChan
}

func New(logger *zap.SugaredLogger, cfg *config2.Config) (tracelistener2.DataProcessor, error) {
	c := cfg.ProcessorConfig

	if c.ProcessorsEnabled == nil {
		c.ProcessorsEnabled = defaultProcessors
	}

	var mp []Module
	var tableSchemas []string

	conn, err := grpc.Dial(cfg.ServiceProviderAddress(), grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("did not connect: %w", err)
	}

	for _, ep := range c.ProcessorsEnabled {
		p, err := processorByName(ep, logger, conn)
		if err != nil {
			return nil, err
		}

		mp = append(mp, p)
		tableSchemas = append(tableSchemas, p.TableSchema())
	}

	logger.Infow("moduleprocessor initialized", "processors", c.ProcessorsEnabled)

	p := Processor{
		chainName:        cfg.ChainName,
		l:                logger,
		writeChan:        make(chan tracelistener2.TraceOperation),
		writebackChan:    make(chan []tracelistener2.WritebackOp),
		errorsChan:       make(chan error),
		moduleProcessors: mp,
		migrations:       tableSchemas,
	}

	go p.lifecycle()

	return &p, nil
}

func (p *Processor) AddModule(m Module) error {
	mn := m.ModuleName()
	for _, em := range p.moduleProcessors {
		if em.ModuleName() == mn {
			return fmt.Errorf("cannot add module %s more than one time", mn)
		}
	}

	p.moduleProcessors = append(p.moduleProcessors, m)

	return nil
}

func processorByName(name string, logger *zap.SugaredLogger, grpcConn *grpc.ClientConn) (Module, error) {
	switch name {
	default:
		return nil, fmt.Errorf("unkonwn Processor %s", name)
	case (&bankProcessor{}).ModuleName():
		return &bankProcessor{
			heightCache: map[bankCacheEntry]models.BalanceRow{},
			l:           logger,
			grpcConn:    grpcConn,
		}, nil
	case (&ibcConnectionsProcessor{}).ModuleName():
		return &ibcConnectionsProcessor{
			connectionsCache: map[connectionCacheEntry]models.IBCConnectionRow{},
			l:                logger,
			grpcConn:         grpcConn,
		}, nil
	case (&delegationsProcessor{}).ModuleName():
		return &delegationsProcessor{
			insertHeightCache: map[delegationCacheEntry]models.DelegationRow{},
			deleteHeightCache: map[delegationCacheEntry]models.DelegationRow{},
			l:                 logger,
			grpcConn:          grpcConn,
		}, nil
	case (&unbondingDelegationsProcessor{}).ModuleName():
		return &unbondingDelegationsProcessor{
			insertHeightCache: map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow{},
			deleteHeightCache: map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow{},
			l:                 logger,
			grpcConn:          grpcConn,
		}, nil
	case (&ibcDenomTracesProcessor{}).ModuleName():
		return &ibcDenomTracesProcessor{
			l:                logger,
			denomTracesCache: map[string]models.IBCDenomTraceRow{},
			grpcConn:         grpcConn,
		}, nil
	case (&ibcChannelsProcessor{}).ModuleName():
		return &ibcChannelsProcessor{
			channelsCache: map[channelCacheEntry]models.IBCChannelRow{},
			l:             logger,
			grpcConn:      grpcConn,
		}, nil
	case (&ibcClientsProcessor{}).ModuleName():
		return &ibcClientsProcessor{
			l:            logger,
			grpcConn:     grpcConn,
			clientsCache: map[clientCacheEntry]models.IBCClientStateRow{},
		}, nil
	case (&authProcessor{}).ModuleName():
		return &authProcessor{
			l:           logger,
			heightCache: map[authCacheEntry]models.AuthRow{},
			grpcConn:    grpcConn,
		}, nil
	case (&validatorsProcessor{}).ModuleName():
		return &validatorsProcessor{
			l:                 logger,
			grpcConn:          grpcConn,
			insertHeightCache: map[validatorCacheEntry]models.ValidatorRow{},
			deleteHeightCache: map[validatorCacheEntry]models.ValidatorRow{},
		}, nil
	}
}

func (p *Processor) Flush() error {
	wb := make([]tracelistener2.WritebackOp, 0, len(p.moduleProcessors))

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

	p.l.Debugw("flush call", "content", wb)

	go func() {
		p.writebackChan <- wb
	}()

	return nil
}

func (p *Processor) lifecycle() {
	for data := range p.writeChan {
		if data.BlockHeight != p.lastHeight && data.BlockHeight != 0 {
			if err := p.Flush(); err != nil {
				p.errorsChan <- fmt.Errorf("error while flushing caches, %w", err)
				continue
			}

			p.l.Infow("processed new block", "height", p.lastHeight)

			p.lastHeight = data.BlockHeight
		}

		for _, mp := range p.moduleProcessors {
			if !mp.OwnsKey(data.Key) {
				continue
			}

			if err := mp.Process(data); err != nil {
				p.errorsChan <- tracelistener2.TracingError{
					InnerError: err,
					Module:     mp.ModuleName(),
					Data:       data,
				}
			}
		}
	}
}
