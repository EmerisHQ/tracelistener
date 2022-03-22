package bulk

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	types2 "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/emerishq/tracelistener/tracelistener/database"
	"github.com/emerishq/tracelistener/tracelistener/processor"
	"github.com/jmoiron/sqlx"
	"golang.org/x/sync/errgroup"

	"github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/store/rootmulti"
	db2 "github.com/tendermint/tm-db"

	"go.uber.org/zap"

	"github.com/emerishq/tracelistener/tracelistener"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

type Importer struct {
	Path         string
	TraceWatcher tracelistener.TraceWatcher
	Processor    tracelistener.DataProcessor
	Logger       *zap.SugaredLogger
	Database     *database.Instance
	Modules      []string
}

func ImportableModulesList() []string {
	ml := make([]string, 0, len(tracelistener.SupportedSDKModuleList))
	for k := range tracelistener.SupportedSDKModuleList {
		ml = append(ml, k.String())
	}

	return ml
}

func (i Importer) validateModulesList() error {
	for _, m := range i.Modules {
		if _, ok := tracelistener.SupportedSDKModuleList[tracelistener.SDKModuleName(m)]; !ok {
			return fmt.Errorf("unknown bulk import module %s", m)
		}
	}

	return nil
}

func (i *Importer) processWritebackData(data []tracelistener.WritebackOp) {
	for _, p := range data {
		if len(p.Data) == 0 {
			continue
		}

		totalUnitsAmt := uint64(0)

		wbUnits := p.SplitStatementToDBLimit()
		for idx, wbUnit := range wbUnits {
			is := wbUnit.InterfaceSlice()

			i.Logger.Infow("writing chunks to database",
				"module", p.SourceModule,
				"total chunks", len(wbUnits),
				"current chunk", idx,
				"total writeback units data", len(wbUnit.Data),
			)

			totalUnitsAmt += uint64(len(wbUnit.Data))

			if err := insertDB(i.Database.Instance.DB, wbUnit.Statement, is); err != nil {
				i.Logger.Errorw("database error",
					"error", err,
					"statement", wbUnit.Statement,
					"type", wbUnit.Type,
					"data", fmt.Sprint(wbUnit.Data),
				)
			}
		}

		i.Logger.Infow("total database rows written",
			"module", p.SourceModule,
			"amount", len(p.Data),
			"chunked amount written", totalUnitsAmt,
			"remains", uint64(len(p.Data))-totalUnitsAmt,
			"equal", uint64(len(p.Data)) == totalUnitsAmt,
		)
	}

	i.Logger.Debugw("finished processing writeback data")
}

func insertDB(db *sqlx.DB, query string, params interface{}) error {
	res, err := db.NamedExec(query, params)
	if err != nil {
		return fmt.Errorf("transaction named exec error, %w", err)
	}

	re, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("transaction named exec error, %w", err)
	}

	if re == 0 {
		return fmt.Errorf("affected rows are zero")
	}

	return nil
}

func (i *Importer) Do() error {
	if err := i.validateModulesList(); err != nil {
		return err
	}

	if i.Modules == nil {
		i.Modules = ImportableModulesList()
	}

	i.Processor.SetDBUpsertEnabled(false)

	i.Path = strings.TrimSuffix(i.Path, ".db")

	dbName := filepath.Base(i.Path)
	path := filepath.Dir(i.Path)

	db, err := db2.NewGoLevelDBWithOpts(dbName, path, &opt.Options{
		ErrorIfMissing: true,
		ReadOnly:       true,
	})

	if err != nil {
		return fmt.Errorf("cannot open chain database, %w", err)
	}

	latestBlockHeight := getLatestVersion(db)

	rm := rootmulti.NewStore(db)
	keys := make([]types2.StoreKey, 0, len(i.Modules))

	for _, ci := range i.Modules {
		key := types.NewKVStoreKey(ci)
		keys = append(keys, key)
		rm.MountStoreWithDB(key, types.StoreTypeIAVL, nil)
	}

	if err := rm.LoadLatestVersion(); err != nil {
		panic(err)
	}

	keysLen := len(keys)

	wbChan := make(chan []tracelistener.WritebackOp, keysLen)

	t0 := time.Now()
	done := make(chan struct{})
	// spawn a goroutine that logs errors from processor's error chan
	go func() {
		for {
			select {
			case <-done:
				return
			case e := <-i.Processor.ErrorsChan():
				te := e.(tracelistener.TracingError)
				i.Logger.Errorw(
					"error while processing data",
					"error", te.InnerError,
					"data", te.Data,
					"moduleName", te.Module)
			case b := <-i.Processor.WritebackChan():
				wbChan <- b
			}
		}
	}()

	processingTime := time.Now()

	eg := errgroup.Group{}
	for idx, key := range keys {
		key := key
		idx := idx
		eg.Go(func() error {
			i.Logger.Infow("processing started", "module", key.Name(), "index", idx+1, "total", keysLen)

			store := rm.GetKVStore(key)
			ii := store.Iterator(nil, nil)

			processedRows := uint64(0)

			for ; ii.Valid(); ii.Next() {
				to := tracelistener.TraceOperation{
					Operation:          tracelistener.WriteOp.String(),
					Key:                ii.Key(),
					Value:              ii.Value(),
					BlockHeight:        uint64(latestBlockHeight),
					SuggestedProcessor: tracelistener.SDKModuleName(key.Name()),
				}

				pp := i.Processor.(*processor.Processor)
				if err := pp.ProcessData(to); err != nil {
					i.Logger.Errorw("processing error", "error", err)
				}

				i.Logger.Debugw("parsed data", "key", string(to.Key), "value", string(to.Value))
				processedRows++
			}

			if err := ii.Error(); err != nil {
				return fmt.Errorf("iterator error, %w", err)
			}

			if err := ii.Close(); err != nil {
				return fmt.Errorf("cannot close iterator, %w", err)
			}

			if err := i.Processor.Flush(); err != nil {
				return fmt.Errorf("cannot flush processor cache, %w", err)
			}

			i.Logger.Infow("processing done", "module", key.Name(), "total_rows", processedRows, "index", idx+1, "total", keysLen)
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	if err := db.Close(); err != nil {
		return fmt.Errorf("database closing error, %w", err)
	}

	/*
		This is dumb but it works.

		What's happening here?
		Since we're importing we know beforehand how many database flushes will happen, hence we know how many times
		the case
		    case b := <-i.Processor.WritebackChan():
		in the processor's select will happen: exactly len(keys) times.

		To make sure we wait until this is true, we wait forever until len(wbChan) is equal to len(keys).
		Since wbChan is a buffered channel with len(keys) elements, the Go runtime gives us synchronization for free.
		After that, we write back data to the database.
	*/
	for len(wbChan) != keysLen {
		runtime.Gosched()
	}

	done <- struct{}{}
	close(wbChan)

	for data := range wbChan {
		i.processWritebackData(data)
	}

	tn := time.Now()
	i.Logger.Infow("import done", "total time", tn.Sub(t0), "processing time", tn.Sub(processingTime))

	return nil
}

// vendored from cosmos-sdk/store/rootmulti/rootmulti.go
const (
	latestVersionKey = "s/latest"
)

func getLatestVersion(db db2.DB) int64 {
	bz, err := db.Get([]byte(latestVersionKey))
	if err != nil {
		panic(err)
	} else if bz == nil {
		return 0
	}

	var latestVersion int64

	if err := gogotypes.StdInt64Unmarshal(&latestVersion, bz); err != nil {
		panic(err)
	}

	return latestVersion
}
