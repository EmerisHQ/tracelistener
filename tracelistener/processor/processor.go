package processor

import (
	"fmt"
	"sync"

	models "github.com/emerishq/demeris-backend-models/tracelistener"

	"github.com/emerishq/tracelistener/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener/config"
	"go.uber.org/zap"
)

type Module interface {
	FlushCache() []tracelistener.WritebackOp
	OwnsKey(key []byte) bool
	Process(data tracelistener.TraceOperation) error
	ModuleName() string
	SDKModuleName() tracelistener.SDKModuleName
	TableSchema() string
}

var defaultProcessors = []string{
	"auth",
	"bank",
	"delegations",
	"unbonding_delegations",
	"ibc_clients",
	"ibc_channels",
	"ibc_connections",
	"ibc_denom_traces",
	"validators",
}

type Processor struct {
	l                *zap.SugaredLogger
	writeChan        chan tracelistener.TraceOperation
	writebackChan    chan []tracelistener.WritebackOp
	errorsChan       chan error
	migrations       []string
	lastHeight       uint64
	chainName        string
	moduleProcessors []Module
	sdkModuleMapping map[tracelistener.SDKModuleName][]Module
	lifecycleStop    chan struct{}

	processingData sync.Mutex
}

func (p *Processor) OpsChan() chan tracelistener.TraceOperation {
	return p.writeChan
}

func (p *Processor) WritebackChan() chan []tracelistener.WritebackOp {
	return p.writebackChan
}

func (p *Processor) DatabaseMigrations() []string {
	return p.migrations
}

func (p *Processor) ErrorsChan() chan error {
	return p.errorsChan
}

func New(logger *zap.SugaredLogger, cfg *config.Config) (tracelistener.DataProcessor, error) {
	c := cfg.Processor

	if c.ProcessorsEnabled == nil {
		c.ProcessorsEnabled = defaultProcessors
	}

	mp := make([]Module, 0)
	tableSchemas := make([]string, 0)

	sdkModuleMapping := map[tracelistener.SDKModuleName][]Module{}

	for _, ep := range c.ProcessorsEnabled {
		p, err := processorByName(ep, logger)
		if err != nil {
			return nil, err
		}

		mp = append(mp, p)
		tableSchemas = append(tableSchemas, p.TableSchema())
		sdkModuleMapping[p.SDKModuleName()] = append(sdkModuleMapping[p.SDKModuleName()], p)
	}

	logger.Infow("processor initialized", "processors", c.ProcessorsEnabled)

	p := Processor{
		chainName:        cfg.ChainName,
		l:                logger,
		writeChan:        make(chan tracelistener.TraceOperation),
		writebackChan:    make(chan []tracelistener.WritebackOp),
		errorsChan:       make(chan error),
		moduleProcessors: mp,
		migrations:       tableSchemas,
		sdkModuleMapping: sdkModuleMapping,
		lifecycleStop:    make(chan struct{}),
	}

	return &p, nil
}

func (p *Processor) StartBackgroundProcessing() {
	go p.lifecycle()
}

func (p *Processor) StopBackgroundProcessing() {
	p.lifecycleStop <- struct{}{}
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

func processorByName(name string, logger *zap.SugaredLogger) (Module, error) {
	switch name {
	default:
		return nil, fmt.Errorf("unknown Processor %s", name)
	case (&bankProcessor{}).ModuleName():
		return &bankProcessor{
			heightCache: map[bankCacheEntry]models.BalanceRow{},
			l:           logger,
		}, nil
	case (&ibcConnectionsProcessor{}).ModuleName():
		return &ibcConnectionsProcessor{
			connectionsCache: map[connectionCacheEntry]models.IBCConnectionRow{},
			l:                logger,
		}, nil
	case (&delegationsProcessor{}).ModuleName():
		return &delegationsProcessor{
			insertHeightCache: map[delegationCacheEntry]models.DelegationRow{},
			deleteHeightCache: map[delegationCacheEntry]models.DelegationRow{},
			l:                 logger,
		}, nil
	case (&unbondingDelegationsProcessor{}).ModuleName():
		return &unbondingDelegationsProcessor{
			insertHeightCache: map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow{},
			deleteHeightCache: map[unbondingDelegationCacheEntry]models.UnbondingDelegationRow{},
			l:                 logger,
		}, nil
	case (&ibcDenomTracesProcessor{}).ModuleName():
		return &ibcDenomTracesProcessor{
			l:                logger,
			denomTracesCache: map[string]models.IBCDenomTraceRow{},
		}, nil
	case (&ibcChannelsProcessor{}).ModuleName():
		return &ibcChannelsProcessor{
			channelsCache: map[channelCacheEntry]models.IBCChannelRow{},
			l:             logger,
		}, nil
	case (&ibcClientsProcessor{}).ModuleName():
		return &ibcClientsProcessor{
			l:            logger,
			clientsCache: map[clientCacheEntry]models.IBCClientStateRow{},
		}, nil
	case (&authProcessor{}).ModuleName():
		return &authProcessor{
			l:           logger,
			heightCache: map[authCacheEntry]models.AuthRow{},
		}, nil
	case (&validatorsProcessor{}).ModuleName():
		return &validatorsProcessor{
			l:                     logger,
			insertValidatorsCache: map[validatorCacheEntry]models.ValidatorRow{},
			deleteValidatorsCache: map[validatorCacheEntry]models.ValidatorRow{},
		}, nil
	}
}

func (p *Processor) Flush() error {
	p.processingData.Lock()
	defer p.processingData.Unlock()
	wb := make([]tracelistener.WritebackOp, 0, len(p.moduleProcessors))

	for _, mp := range p.moduleProcessors {
		cd := mp.FlushCache()
		for _, entry := range cd {
			if len(entry.Data) == 0 {
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
	for {
		select {
		case <-p.lifecycleStop:
			return
		case data := <-p.writeChan:
			if data.BlockHeight != p.lastHeight && data.BlockHeight != 0 {
				if err := p.Flush(); err != nil {
					p.errorsChan <- fmt.Errorf("error while flushing caches, %w", err)
					continue
				}

				p.l.Infow("processed new block", "height", p.lastHeight)

				p.lastHeight = data.BlockHeight
			}

			if err := p.ProcessData(data); err != nil {
				p.errorsChan <- fmt.Errorf("error while flushing caches, %w", err)
				continue
			}
		}
	}
}

func (p *Processor) ProcessData(data tracelistener.TraceOperation) error {
	processorList := p.moduleProcessors

	if data.SuggestedProcessor != "" {
		p.l.Debugw("suggested processor", "name", data.SuggestedProcessor)
		// only consider the associated processors
		plist, ok := p.sdkModuleMapping[data.SuggestedProcessor]
		if ok {
			p.l.Debugw("found processor matching against sdk module mapping", "requested module", data.SuggestedProcessor)
			processorList = plist
		}
	}

	p.processData(processorList, data)

	return nil
}

// TODO: error group?
func (p *Processor) processData(processorList []Module, data tracelistener.TraceOperation) {
	p.processingData.Lock()
	defer p.processingData.Unlock()
	for _, mp := range processorList {
		if !mp.OwnsKey(data.Key) {
			continue
		}

		mn := mp.ModuleName()
		// Log line used to trigger Grafana alerts.
		// Do not modify or remove without changing the corresponding dashboards
		if data.SuggestedProcessor == "" {
			// log this only when running non in bulk import mode
			p.l.Infow("Probe", "c", "gaia", "n", mn)
		}

		if err := mp.Process(data); err != nil {
			p.errorsChan <- tracelistener.TracingError{
				InnerError: err,
				Module:     mp.ModuleName(),
				Data:       data,
			}
		}
	}
}
