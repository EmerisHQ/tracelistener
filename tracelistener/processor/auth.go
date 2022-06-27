package processor

import (
	"bytes"
	"sync"

	"github.com/emerishq/tracelistener/models"
	"github.com/emerishq/tracelistener/tracelistener"
	"github.com/emerishq/tracelistener/tracelistener/processor/datamarshaler"
	"github.com/emerishq/tracelistener/tracelistener/tables"
	"go.uber.org/zap"
)

var authTable = tables.NewAuthTable("tracelistener.auth")

type authCacheEntry struct {
	address   string
	accNumber uint64
}

type authProcessor struct {
	l           *zap.SugaredLogger
	heightCache map[authCacheEntry]models.AuthRow
	m           sync.Mutex
}

func (*authProcessor) Migrations() []string {
	return []string{authTable.CreateTable()}
}

func (b *authProcessor) ModuleName() string {
	return "auth"
}

func (b *authProcessor) SDKModuleName() tracelistener.SDKModuleName {
	return tracelistener.Acc
}

func (b *authProcessor) UpsertStatement() string {
	return authTable.Upsert()
}

func (b *authProcessor) InsertStatement() string {
	return authTable.Insert()
}

func (b *authProcessor) DeleteStatement() string {
	panic("auth processor never deletes")
}

func (b *authProcessor) FlushCache() []tracelistener.WritebackOp {
	b.m.Lock()
	defer b.m.Unlock()
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
			Type: tracelistener.Write,
			Data: l,
		},
	}
}

func (b *authProcessor) OwnsKey(key []byte) bool {
	return bytes.HasPrefix(key, datamarshaler.AuthKey)
}

func (b *authProcessor) Process(data tracelistener.TraceOperation) error {
	b.m.Lock()
	defer b.m.Unlock()

	res, err := datamarshaler.NewDataMarshaler(b.l).Auth(data)
	if err != nil {
		return err
	}

	b.heightCache[authCacheEntry{
		address:   res.Address,
		accNumber: res.AccountNumber,
	}] = res

	return nil
}
