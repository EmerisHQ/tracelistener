package bulk

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/allinbits/tracelistener/tracelistener/database"
	types2 "github.com/cosmos/cosmos-sdk/store/types"
	"golang.org/x/sync/errgroup"

	"github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/store/rootmulti"
	db2 "github.com/tendermint/tm-db"

	"go.uber.org/zap"

	"github.com/allinbits/tracelistener/tracelistener"
	gogotypes "github.com/gogo/protobuf/types"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

type counter struct {
	ctr int
	m   sync.Mutex
}

func (c *counter) increment() {
	c.m.Lock()
	defer c.m.Unlock()
	c.ctr++
}

func (c *counter) value() int {
	c.m.Lock()
	defer c.m.Unlock()
	return c.ctr
}

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

func (i *Importer) processWritebackData(data []tracelistener.WritebackOp, dbMutex *sync.Mutex, ctr *counter) {
	i.Logger.Info("requesting database lock for writing...")
	dbMutex.Lock()
	defer func() {
		ctr.increment()
		i.Logger.Info("releasing database lock now!")
		dbMutex.Unlock()
	}()

	i.Logger.Info("lock acquired, proceeding with database write!")
	for _, p := range data {
		if len(p.Data) == 0 {
			continue
		}

		totalUnitsAmt := uint64(0)

		wbUnits := p.SplitStatementToDBLimit()
		for idx, wbUnit := range wbUnits {
			is := wbUnit.InterfaceSlice()

			i.Logger.Infow("writing chunks to database",
				"total chunks", len(wbUnits),
				"current chunk", idx,
				"total writeback units data", len(wbUnit.Data),
			)

			totalUnitsAmt += uint64(len(wbUnit.Data))

			if err := i.Database.Add(wbUnit.DatabaseExec, is); err != nil {
				i.Logger.Error("database error ", err)
			}
		}

		i.Logger.Infow("total database rows to be written",
			"amount", len(p.Data),
			"chunked amount written", totalUnitsAmt,
			"remains", uint64(len(p.Data))-totalUnitsAmt,
			"equal", uint64(len(p.Data)) == totalUnitsAmt,
		)
	}

	i.Logger.Debugw("finished processing writeback data")
}

func (i *Importer) Do() error {
	if err := i.validateModulesList(); err != nil {
		return err
	}

	if i.Modules == nil {
		i.Modules = ImportableModulesList()
	}

	dbMutex := sync.Mutex{}
	ctr := &counter{}
	t0 := time.Now()
	// spawn a goroutine that logs errors from processor's error chan
	go func() {
		for {
			select {
			case e := <-i.Processor.ErrorsChan():
				te := e.(tracelistener.TracingError)
				i.Logger.Errorw(
					"error while processing data",
					"error", te.InnerError,
					"data", te.Data,
					"moduleName", te.Module)
			case b := <-i.Processor.WritebackChan():
				i.Logger.Debugw("wbchan called", "idx", ctr.value())
				i.processWritebackData(b, &dbMutex, ctr)
			}
		}
	}()

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

	processingTime := time.Now()

	keysLen := len(keys)

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

				if err := i.TraceWatcher.ParseOperation(to); err != nil {
					return fmt.Errorf("cannot parse operation %v, %w", to, err)
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

			time.Sleep(1 * time.Second)

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

	i.Logger.Info("requesting database lock before finalizing bulk import...")

	/*
		This is dumb but it works.

		What's happening here?
		Since we're importing we know beforehand how many database flushes will happen, hence we know how many times
		the case
		    case b := <-i.Processor.WritebackChan():
		in the processor's select will happen: exactly len(keys) times.

		To make sure we wait until this is true, we wait forever until dbWritebackCallAmt is equal to len(keys).
		To further strengthen our logic here we only increment dbWritebackCallAmt when the WritebackChan acquires a lock
		on dbMutex, so that when this infinite for cycle will actually break, we block again until the database func call has
		finished writing.
		After that, we acquire the lock and continue with our own way.
	*/
	for ctr.value() != keysLen {
		runtime.Gosched()
	}

	dbMutex.Lock()
	i.Logger.Info("database lock acquired, finalizing")
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
