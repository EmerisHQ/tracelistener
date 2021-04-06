package gaia_processor

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/simapp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/allinbits/tracelistener"
	"go.uber.org/zap"
)

type balanceWritebackPacket struct {
	ID          uint64 `db:"id" json:"-"`
	Address     string `db:"address" json:"address"`
	Amount      uint64 `db:"amount" json:"amount"`
	Denom       string `db:"denom" json:"denom"`
	BlockHeight uint64 `db:"height" json:"block_height"`
}

type cacheEntry struct {
	address string
	denom   string
}

var p processor

type processor struct {
	l             *zap.SugaredLogger
	writeChan     chan tracelistener.TraceOperation
	writebackChan chan tracelistener.WritebackOp
	cdc           codec.Marshaler
	lastHeight    uint64
	heightCache   map[cacheEntry]balanceWritebackPacket
}

func New(logger *zap.SugaredLogger) (tracelistener.DataProcessorInfos, error) {
	p = processor{
		l:             logger,
		writeChan:     make(chan tracelistener.TraceOperation),
		writebackChan: make(chan tracelistener.WritebackOp),
		heightCache:   map[cacheEntry]balanceWritebackPacket{},
	}

	cdc, _ := simapp.MakeCodecs()
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

func (p *processor) flushCache() []interface{} {
	if len(p.heightCache) == 0 {
		return nil
	}

	l := make([]interface{}, 0, len(p.heightCache))

	for _, v := range p.heightCache {
		l = append(l, v)
	}

	p.heightCache = map[cacheEntry]balanceWritebackPacket{}

	return l
}

func (p *processor) lifecycle() {
	for data := range p.writeChan {
		switch {
		case bytes.HasPrefix(data.Key, types.BalancesPrefix): // balances
			if data.BlockHeight != p.lastHeight && data.BlockHeight != 0 {
				p.writebackChan <- tracelistener.WritebackOp{
					DatabaseExec: insertBalanceQuery,
					Data:         p.flushCache(),
				}

				p.l.Infow("processed new block", "height", p.lastHeight)

				p.lastHeight = data.BlockHeight
			}
			addrBytes := data.Key
			pLen := len(types.BalancesPrefix)
			addr := addrBytes[pLen : pLen+20]

			coins := sdk.Coin{}

			if err := p.cdc.UnmarshalBinaryBare(data.Value, &coins); err != nil {
				// TODO: handle this
				fmt.Println(err)
			}

			if coins.Amount.IsNil() || coins.IsZero() || !coins.IsValid() {
				continue
			}

			hAddr := hex.EncodeToString(addr)
			p.heightCache[cacheEntry{
				address: hAddr,
				denom:   coins.Denom,
			}] = balanceWritebackPacket{
				Address:     hAddr,
				Amount:      coins.Amount.Uint64(),
				Denom:       coins.Denom,
				BlockHeight: data.BlockHeight,
			}
		}
	}
}
