package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/allinbits/tracelistener/tracelistener"
)

type traceInfo struct {
	BlockHeight uint64
	KeyLength   uint64
	ValueLength uint64
	Length      uint64
}

type traceInfos []traceInfo

func (ti traceInfos) CSV() [][]string {
	ret := make([][]string, 0, 1+len(ti)) // add 1 row for title

	ret = append(ret, []string{"block_height", "key_lengt", "value_length", "length"})

	for _, t := range ti {
		ret = append(ret, []string{
			strconv.FormatUint(t.BlockHeight, 10),
			strconv.FormatUint(t.KeyLength, 10),
			strconv.FormatUint(t.ValueLength, 10),
			strconv.FormatUint(t.Length, 10),
		})
	}

	return ret
}

func main() {
	fname := os.Args[1]

	rows, err := loadTestFile(fname)
	if err != nil {
		panic(err)
	}

	ti, err := getTraceInfo(rows)
	if err != nil {
		panic(err)
	}

	o, err := os.OpenFile("tracestats.csv", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0755)
	if err != nil {
		panic(err)
	}

	w := csv.NewWriter(o)
	for _, record := range ti.CSV() {
		if err := w.Write(record); err != nil {
			panic(err)
		}
	}

	o.Close()
}

func getTraceInfo(traces []string) (traceInfos, error) {
	ret := make([]traceInfo, 0, len(traces))

	for _, t := range traces {
		tr := tracelistener.TraceOperation{}
		if err := json.Unmarshal([]byte(t), &tr); err != nil {
			return nil, err
		}

		ret = append(ret, traceInfo{
			BlockHeight: tr.Metadata.BlockHeight,
			KeyLength:   uint64(len(tr.Key)),
			ValueLength: uint64(len(tr.Value)),
			Length:      uint64(len(t)),
		})
	}

	return ret, nil
}

func loadTestFile(fname string) ([]string, error) {
	file, err := os.Open(fname)
	if err != nil {
		return nil, fmt.Errorf("cannot open file %s, %w", fname, err)
	}

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 1000000) // a very high capacity
	scanner.Buffer(buf, 1000000)

	ret := []string{}
	for scanner.Scan() {
		ret = append(ret, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning error, %w", err)
	}

	return ret, nil
}
