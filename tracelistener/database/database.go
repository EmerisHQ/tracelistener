package database

import (
	dbutils "github.com/emerishq/emeris-utils/database"
	"math/rand"
	"time"
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
	return i.Instance.Exec(query, data, nil)
}

// Jitter takes a duration <param: delta> and a dividing factor <param: factor>
// and returns a func that makes the current thread sleep for a random amount
// in range delta/[1...factor]
//
// Ex: delta: 10s     factor: 10   => sleep can be anything from  1 sec to 10 sec.
//     delta: 10s     factor: 20   => sleep can be anything from .5 sec to 10 sec.
//     delta: 10s     factor: 100  => sleep can be anything from .1 sec to 10 sec.
func Jitter(delta time.Duration, factor int) func() {
	return func() {
		if delta <= 0 {
			return
		}
		if factor <= 0 || factor >= 100 {
			return
		}
		if delta < time.Millisecond*10 {
			delta = time.Millisecond * 10
		}
		rand.Seed(time.Now().Unix())
		factor = rand.Intn(factor) + 1
		time.Sleep(delta / time.Duration(factor))
	}
}
