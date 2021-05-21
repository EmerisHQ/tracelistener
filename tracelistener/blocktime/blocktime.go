package blocktime

import (
	"context"
	"fmt"
	"time"

	"github.com/allinbits/demeris-backend/models"

	"github.com/tendermint/tendermint/types"

	coretypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/tendermint/tendermint/rpc/client/http"

	"github.com/allinbits/demeris-backend/utils/database"
	"go.uber.org/zap"
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
	INSERT INTO tracelistener.blocktime 
		(chain_name, block_time) 
	VALUES 
		(:chain_name, :block_time) 
	ON CONFLICT
		(chain_name)
	DO UPDATE SET 
		chain_name=EXCLUDED.chain_name,
		block_time=EXCLUDED.block_time;
	`
)

type blockTimeObject struct {
	models.TracelistenerDatabaseRow

	BlockTime time.Time `db:"block_time"`
}

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

func (w *Watcher) lifecycle() {
	for data := range w.tm {
		block, ok := data.Data.(types.EventDataNewBlock)
		if !ok {
			w.l.Errorw("tried casting data to EventDataNewBlock", "real type", fmt.Sprintf("%T", block))
			continue
		}

		w.l.Debugw("new block", "block", block)

		if block.Block == nil {
			continue
		}

		if err := w.insertBlockTime(blockTimeObject{
			TracelistenerDatabaseRow: models.TracelistenerDatabaseRow{
				ChainName: w.chainName,
			},
			BlockTime: block.Block.Time,
		}); err != nil {
			w.l.Errorw("cannot insert block time", "chain", w.chainName, "error", err)
		}
	}
}

func (w *Watcher) insertBlockTime(blo blockTimeObject) error {
	return w.di.Exec(insertBlocktime, blo, nil)
}
