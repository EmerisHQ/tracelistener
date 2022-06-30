package database

import (
	"math/rand"
	"time"

	dbutils "github.com/emerishq/tracelistener/database"
)

type Instance struct {
	Instance   *dbutils.Instance
	connString string
}

func New(connString string) (*Instance, error) {

	i, err := dbutils.New(connString)

	if err != nil {
		return nil, err
	}

	ii := &Instance{
		Instance:   i,
		connString: connString,
	}

	ii.runMigrations()

	return ii, nil
}

func (i *Instance) Add(query string, data []interface{}, sleepFunc func()) error {
	if sleepFunc != nil {
		sleepFunc()
	}
	_, err := i.Instance.DB.NamedExec(query, data)
	return err
}

// Jitter takes a duration <param: delta> and a dividing factor <param: factor>
// and returns a func that makes the current thread sleep for a random amount
// in range delta/[1...factor]
//
// Ex: delta: 10s     factor: 10   => sleep can be anything from  1 sec to 10 sec.
//     delta: 10s     factor: 20   => sleep can be anything from .5 sec to 10 sec.
//     delta: 10s     factor: 100  => sleep can be anything from .1 sec to 10 sec.
//
// Min sleep time is 10MS. We are implementing something similar to equal jitter
// (as we don't have backoff, there's no memory). But following the philosophy of
// equal jitter and not allowing a very small sleep time.
func Jitter(delta time.Duration, factor int) func() {
	return func() {
		if delta <= 0 || factor <= 0 {
			return
		}
		rand.Seed(time.Now().Unix())
		factor = rand.Intn(factor) + 1 //nolint:gosec
		delta /= time.Duration(factor)
		if delta < time.Millisecond*10 {
			delta = time.Millisecond * 10
		}
		time.Sleep(delta)
	}
}
