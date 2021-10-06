package moduleprocessor_test

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	tracelistener2 "github.com/allinbits/tracelistener"
	config2 "github.com/allinbits/tracelistener/config"
	gaia_processor2 "github.com/allinbits/tracelistener/moduleprocessor"

	"go.uber.org/zap"

	"github.com/stretchr/testify/require"
)

type dumbModule struct {
	wbOp          []tracelistener2.WritebackOp
	key           []byte
	alwaysOwnsKey bool
	processFunc   func(data tracelistener2.TraceOperation) error
	tableSchema   string
	moduleName    string
}

func (d dumbModule) FlushCache() []tracelistener2.WritebackOp {
	return d.wbOp
}

func (d dumbModule) OwnsKey(key []byte) bool {
	if d.alwaysOwnsKey {
		return true
	}

	return bytes.Equal(key, d.key)
}

func (d dumbModule) Process(data tracelistener2.TraceOperation) error {
	return d.processFunc(data)
}

func (d dumbModule) ModuleName() string {
	if d.moduleName != "" {
		return d.moduleName
	}
	return "dumbModule"
}

func (d dumbModule) TableSchema() string {
	return d.tableSchema
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config2.Config
		wantErr bool
	}{
		{
			"nonexistant Processor type",
			&config2.Config{
				ProcessorConfig: config2.ProcessorConfig{
					ProcessorsEnabled: []string{"doesn't exists"},
				},
			},
			true,
		},
		{
			"no gaia config specified, default list of processors enabled",
			&config2.Config{},
			false,
		},
		{
			"gaia config specified with a list of processors enabled",
			&config2.Config{
				ProcessorConfig: config2.ProcessorConfig{
					ProcessorsEnabled: []string{"bank"},
				},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := gaia_processor2.New(zap.NewNop().Sugar(), tt.cfg)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
		})
	}
}

func TestLifecycle(t *testing.T) {
	tests := []struct {
		name            string
		presentMessages []tracelistener2.TraceOperation
		newMessage      tracelistener2.TraceOperation
		processorFunc   func(data tracelistener2.TraceOperation) error
		wantErr         bool
		shouldSendWb    bool
	}{
		{
			"no error when queueing new message accepted by the processor",
			nil,
			tracelistener2.TraceOperation{
				Operation:   string(tracelistener2.WriteOp),
				Key:         []byte("key"),
				Value:       []byte("key"),
				BlockHeight: 0,
			},
			func(_ tracelistener2.TraceOperation) error {
				return nil
			},
			false,
			false,
		},
		{
			"error when queueing new message accepted by the processor",
			nil,
			tracelistener2.TraceOperation{
				Operation:   string(tracelistener2.WriteOp),
				Key:         []byte("key"),
				Value:       []byte("key"),
				BlockHeight: 0,
			},
			func(_ tracelistener2.TraceOperation) error {
				return fmt.Errorf("oh no, error")
			},
			true,
			false,
		},
		{
			"new message, block different re: last height",
			[]tracelistener2.TraceOperation{
				{
					Operation:   string(tracelistener2.WriteOp),
					Key:         []byte("key"),
					Value:       []byte("key"),
					BlockHeight: 0,
				},
			},
			tracelistener2.TraceOperation{
				Operation:   string(tracelistener2.WriteOp),
				Key:         []byte("key"),
				Value:       []byte("key"),
				BlockHeight: 1,
			},
			func(_ tracelistener2.TraceOperation) error {
				return nil
			},
			false,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := gaia_processor2.New(
				zap.NewNop().Sugar(),
				&config2.Config{},
			)

			require.NoError(t, err)
			require.NotNil(t, p)

			// we know p is of type moduleprocessor.Processor
			gp := p.(*gaia_processor2.Processor)
			require.NotNil(t, gp)

			// let's add something we can actually control
			dumb := dumbModule{
				processFunc:   tt.processorFunc,
				moduleName:    "dumb",
				alwaysOwnsKey: true,
				wbOp:          nil,
			}

			require.NoError(t, gp.AddModule(dumb))

			for _, present := range tt.presentMessages {
				p.OpsChan() <- present

				var receivedErr error
				require.Never(t, func() bool {
					receivedErr = <-p.ErrorsChan()
					return nil != receivedErr
				}, 1*time.Second, 500*time.Millisecond, "received error: %s", receivedErr)
			}

			go func() {
				p.OpsChan() <- tt.newMessage
			}()

			if tt.wantErr {
				// we get an error on errorschan if something goes bad
				require.Eventually(t, func() bool {
					return <-p.ErrorsChan() != nil
				}, 10*time.Second, 500*time.Millisecond)

				return
			}

			var receivedErr error
			require.Never(t, func() bool {
				receivedErr = <-p.ErrorsChan()
				return nil != receivedErr
			}, 10*time.Second, 500*time.Millisecond, "received error: %s", receivedErr)

			if tt.shouldSendWb {
				require.Eventually(t, func() bool {
					return <-p.WritebackChan() != nil
				}, 10*time.Second, 500*time.Millisecond)

				return
			}
		})
	}
}

func TestProcessor_AddModule(t *testing.T) {
	tests := []struct {
		name            string
		existingModules []gaia_processor2.Module
		newModule       gaia_processor2.Module
		wantErr         bool
	}{
		{
			"no existing modules, no error",
			nil,
			dumbModule{},
			false,
		},
		{
			"existing modules, new module does not conflict",
			[]gaia_processor2.Module{
				dumbModule{
					moduleName: "dumbModuleTwo",
				},
			},
			dumbModule{},
			false,
		},
		{
			"existing modules, new module does conflict",
			[]gaia_processor2.Module{
				dumbModule{},
			},
			dumbModule{},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &gaia_processor2.Processor{}

			for _, em := range tt.existingModules {
				require.NoError(t, p.AddModule(em))
			}

			err := p.AddModule(tt.newModule)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
