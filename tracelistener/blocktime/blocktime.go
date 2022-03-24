package blocktime

import (
	"context"
	"fmt"

	"github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"
	"go.uber.org/zap"

	models "github.com/emerishq/demeris-backend-models/tracelistener"
	"github.com/emerishq/emeris-utils/database"
)

const (
	tendermintWSPort = 26657
	blEvents         = "tm.event='NewBlock'"

	CreateTable = `CREATE TABLE IF NOT EXISTS tracelistener.blocktime (
		id serial unique primary key,
		chain_name text not null,
		block_time timestamp not null,
		unique(chain_name)
	)`

	insertBlocktime = `
	INSERT INTO tracelistener.blocktime as tb
		(chain_name, block_time) 
	VALUES 
		(:chain_name, :block_time) 
	ON CONFLICT
		(chain_name)
	DO UPDATE SET 
		chain_name=EXCLUDED.chain_name,
		block_time=EXCLUDED.block_time
		WHERE EXCLUDED.block_time > tb.block_time;
	`
)

type Watcher struct {
	di        *database.Instance
	chainName string
	l         *zap.SugaredLogger
	tm        <-chan coretypes.ResultEvent
}

func New(di *database.Instance, chainName string, l *zap.SugaredLogger) *Watcher {
	return &Watcher{
		di:        di,
		chainName: chainName,
		l:         l,
	}
}

func (w *Watcher) Connect() error {
	wsc, err := http.New(fmt.Sprintf("http://%s:%d", w.chainName, tendermintWSPort), "/websocket")
	if err != nil {
		return err
	}

	if err := wsc.Start(); err != nil {
		return err
	}

	resChan, err := wsc.Subscribe(context.Background(), "tracelistener", blEvents)
	if err != nil {
		return err
	}

	w.tm = resChan

	go w.lifecycle()

	return nil
}

func (w *Watcher) ParseBlockData(data coretypes.ResultEvent) error {
	block, ok := data.Data.(types.EventDataNewBlock)
	if !ok {
		return fmt.Errorf("tried casting data to EventDataNewBlock, real type %T", block)
	}

	w.l.Debugw("new block", "block", block)

	// Log line used to trigger Grafana alerts.
	// Do not modify or remove without changing the corresponding dashboards
	w.l.Infow("Probe", "c", "block")

	if block.Block == nil {
		return nil
	}

	if err := w.insertBlockTime(models.BlockTimeRow{
		TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
			ChainName: w.chainName,
		},
		BlockTime: block.Block.Time,
	}); err != nil && err.Error() != "affected rows are zero" {
		return fmt.Errorf("cannot insert block time, %w", err)
	}

	return nil
}

func (w *Watcher) lifecycle() {
	for data := range w.tm {
		if err := w.ParseBlockData(data); err != nil {
			w.l.Errorw("cannot parse block data", "error", err)
		}
	}
}

func (w *Watcher) insertBlockTime(blo models.BlockTimeRow) error {
	return w.di.Exec(insertBlocktime, blo, nil)
}
