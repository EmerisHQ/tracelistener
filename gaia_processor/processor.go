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

const insertQuery = `
UPSERT INTO tracelistener.balances (address, amount, denom, height) VALUES (:address, :amount, :denom, :height)
`

var p processor

type processor struct {
	l             *zap.SugaredLogger
	writeChan     chan tracelistener.TraceOperation
	writebackChan chan []interface{}
	cdc           codec.Marshaler
	lastHeight    uint64
	heightCache   map[string]GaiaWritebackPacket
}

type GaiaWritebackPacket struct {
	Address     string `db:"address"`
	Amount      uint64 `db:"amount"`
	Denom       string `db:"denom"`
	BlockHeight uint64 `db:"height"`
}

func New(logger *zap.SugaredLogger) (tracelistener.DataProcessorInfos, error) {
	p = processor{
		l:             logger,
		writeChan:     make(chan tracelistener.TraceOperation),
		writebackChan: make(chan []interface{}),
		heightCache:   map[string]GaiaWritebackPacket{},
	}

	cdc, _ := simapp.MakeCodecs()
	p.cdc = cdc

	go p.lifecycle()

	return tracelistener.DataProcessorInfos{
		OpsChan:         p.writeChan,
		WritebackChan:   p.writebackChan,
		InsertQueryTmpl: insertQuery,
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

	p.heightCache = map[string]GaiaWritebackPacket{}

	return l
}

func (p *processor) lifecycle() {
	for data := range p.writeChan {
		if bytes.HasPrefix(data.Key, types.BalancesPrefix) {
			if data.BlockHeight != p.lastHeight && data.BlockHeight != 0 {
				p.writebackChan <- p.flushCache()

				p.l.Infow("processed new block", "height", p.lastHeight)

				p.lastHeight = data.BlockHeight
			}
			addrBytes := data.Key
			pLen := len(types.BalancesPrefix)
			addr := addrBytes[pLen : pLen+20]

			coins := sdk.Coin{}

			if err := p.cdc.UnmarshalBinaryBare(data.Value, &coins); err != nil {
				fmt.Println(err)
			}

			if coins.Amount.IsNil() || coins.IsZero() || !coins.IsValid() {
				continue
			}

			hAddr := hex.EncodeToString(addr)
			p.heightCache[hAddr] = GaiaWritebackPacket{
				Address:     hAddr,
				Amount:      coins.Amount.Uint64(),
				Denom:       coins.Denom,
				BlockHeight: data.BlockHeight,
			}
		}
	}
}
