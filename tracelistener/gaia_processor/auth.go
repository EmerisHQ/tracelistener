package gaia_processor

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/allinbits/demeris-backend/models"
	"github.com/allinbits/demeris-backend/tracelistener"
	types3 "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	"go.uber.org/zap"
)

type authCacheEntry struct {
	address        string
	sequenceNumber uint64
	accNumber      uint64
}

type authProcessor struct {
	l           *zap.SugaredLogger
	heightCache map[authCacheEntry]models.AuthRow
}

func (*authProcessor) TableSchema() string {
	return createAuthTable
}

func (b *authProcessor) ModuleName() string {
	return "auth"
}

func (b *authProcessor) FlushCache() []tracelistener.WritebackOp {
	if len(b.heightCache) == 0 {
		return nil
	}

	l := make([]models.DatabaseEntrier, 0, len(b.heightCache))

	for _, v := range b.heightCache {
		l = append(l, v)
	}

	b.heightCache = map[authCacheEntry]models.AuthRow{}

	return []tracelistener.WritebackOp{
		{
			DatabaseExec: insertAuth,
			Data:         l,
		},
	}
}

func (b *authProcessor) OwnsKey(key []byte) bool {
	return bytes.HasPrefix(key, types.AddressStoreKeyPrefix)
}

func (b *authProcessor) Process(data tracelistener.TraceOperation) error {
	if len(data.Key) != types3.AddrLen+1 {
		// key len must be len(account bytes) + 1
		return nil
	}

	var acc types.AccountI

	if err := p.cdc.UnmarshalInterface(data.Value, &acc); err != nil {
		// HACK: since slashing and auth use the same prefix for two different things,
		// let's ignore "no concrete type registered for type URL *" errors.
		// This is ugly, but frankly this is the only way to do it.
		// Frojdi please bless us with the new SDK ASAP.

		if strings.HasPrefix(err.Error(), "no concrete type registered for type URL") {
			return nil
		}

		return err
	}

	baseAcc, ok := acc.(*types.BaseAccount)
	if !ok {
		return fmt.Errorf("cannot cast account to BaseAccount, type %T", baseAcc)
	}

	_, bz, err := bech32.DecodeAndConvert(baseAcc.Address)
	if err != nil {
		return fmt.Errorf("cannot parse %s as bech32, %w", baseAcc.Address, err)
	}

	hAddr := hex.EncodeToString(bz)
	b.l.Debugw("new auth store write",
		"operation", data.Operation,
		"address", hAddr,
		"sequence_number", acc.GetSequence(),
		"account_number", acc.GetAccountNumber(),
		"height", data.BlockHeight,
		"txHash", data.TxHash,
	)

	b.heightCache[authCacheEntry{
		address:        hAddr,
		sequenceNumber: acc.GetSequence(),
		accNumber:      acc.GetAccountNumber(),
	}] = models.AuthRow{
		Address:        hAddr,
		SequenceNumber: acc.GetSequence(),
		AccountNumber:  acc.GetAccountNumber(),
	}

	return nil
}
