package evm

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ocr2keepers "github.com/smartcontractkit/ocr2keepers/pkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	commonmocks "github.com/smartcontractkit/chainlink/v2/common/mocks"
	"github.com/smartcontractkit/chainlink/v2/core/chains/evm/logpoller"
	"github.com/smartcontractkit/chainlink/v2/core/chains/evm/logpoller/mocks"
	evmtypes "github.com/smartcontractkit/chainlink/v2/core/chains/evm/types"
	"github.com/smartcontractkit/chainlink/v2/core/utils"
)

func TestGetActiveUpkeepIDs(t *testing.T) {
	tests := []struct {
		Name         string
		LatestHead   int64
		ActiveIDs    []string
		ExpectedErr  error
		ExpectedKeys []ocr2keepers.UpkeepIdentifier
	}{
		{Name: "NoActiveIDs", LatestHead: 1, ActiveIDs: []string{}, ExpectedKeys: []ocr2keepers.UpkeepIdentifier{}},
		{Name: "AvailableActiveIDs", LatestHead: 1, ActiveIDs: []string{"8", "9", "3", "1"}, ExpectedKeys: []ocr2keepers.UpkeepIdentifier{
			ocr2keepers.UpkeepIdentifier("8"),
			ocr2keepers.UpkeepIdentifier("9"),
			ocr2keepers.UpkeepIdentifier("3"),
			ocr2keepers.UpkeepIdentifier("1"),
		}},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			actives := make(map[string]activeUpkeep)
			for _, id := range test.ActiveIDs {
				idNum := big.NewInt(0)
				idNum.SetString(id, 10)
				actives[id] = activeUpkeep{ID: idNum}
			}

			mht := commonmocks.NewHeadTracker[*evmtypes.Head, common.Hash](t)

			rg := &EvmRegistry{
				HeadProvider: HeadProvider{
					ht: mht,
				},
				active: actives,
			}

			keys, err := rg.GetActiveUpkeepIDs(context.Background())

			if test.ExpectedErr != nil {
				assert.ErrorIs(t, err, test.ExpectedErr)
			} else {
				assert.Nil(t, err)
			}

			if len(test.ExpectedKeys) > 0 {
				for _, key := range keys {
					assert.Contains(t, test.ExpectedKeys, key)
				}
			} else {
				assert.Equal(t, test.ExpectedKeys, keys)
			}
		})
	}
}

func TestGetActiveUpkeepIDsByType(t *testing.T) {
	tests := []struct {
		Name         string
		LatestHead   int64
		ActiveIDs    []string
		ExpectedErr  error
		ExpectedKeys []ocr2keepers.UpkeepIdentifier
		Triggers     []uint8
	}{
		{Name: "no active ids", LatestHead: 1, ActiveIDs: []string{}, ExpectedKeys: []ocr2keepers.UpkeepIdentifier{}},
		{
			Name:       "get log upkeeps",
			LatestHead: 1,
			ActiveIDs:  []string{"8", "32329108151019397958065800113404894502874153543356521479058624064899121404671"},
			ExpectedKeys: []ocr2keepers.UpkeepIdentifier{
				ocr2keepers.UpkeepIdentifier("32329108151019397958065800113404894502874153543356521479058624064899121404671"),
			},
			Triggers: []uint8{uint8(logTrigger)},
		},
		{
			Name:       "get conditional upkeeps",
			LatestHead: 1,
			ActiveIDs:  []string{"8", "32329108151019397958065800113404894502874153543356521479058624064899121404671"},
			ExpectedKeys: []ocr2keepers.UpkeepIdentifier{
				ocr2keepers.UpkeepIdentifier("8"),
			},
			Triggers: []uint8{uint8(conditionTrigger)},
		},
		{
			Name:       "get multiple types of upkeeps",
			LatestHead: 1,
			ActiveIDs:  []string{"8", "32329108151019397958065800113404894502874153543356521479058624064899121404671"},
			ExpectedKeys: []ocr2keepers.UpkeepIdentifier{
				ocr2keepers.UpkeepIdentifier("8"),
				ocr2keepers.UpkeepIdentifier("32329108151019397958065800113404894502874153543356521479058624064899121404671"),
			},
			Triggers: []uint8{uint8(logTrigger), uint8(conditionTrigger)},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			actives := make(map[string]activeUpkeep)
			for _, id := range test.ActiveIDs {
				idNum := big.NewInt(0)
				idNum.SetString(id, 10)
				actives[id] = activeUpkeep{ID: idNum}
			}

			mht := commonmocks.NewHeadTracker[*evmtypes.Head, common.Hash](t)

			rg := &EvmRegistry{
				HeadProvider: HeadProvider{
					ht: mht,
				},
				active: actives,
			}

			keys, err := rg.GetActiveUpkeepIDsByType(context.Background(), test.Triggers...)

			if test.ExpectedErr != nil {
				assert.ErrorIs(t, err, test.ExpectedErr)
			} else {
				assert.Nil(t, err)
			}

			if len(test.ExpectedKeys) > 0 {
				for _, key := range keys {
					assert.Contains(t, test.ExpectedKeys, key)
				}
			} else {
				assert.Equal(t, test.ExpectedKeys, keys)
			}
		})
	}
}

func TestPollLogs(t *testing.T) {
	tests := []struct {
		Name             string
		LastPoll         int64
		Address          common.Address
		ExpectedLastPoll int64
		ExpectedErr      error
		LatestBlock      *struct {
			OutputBlock int64
			OutputErr   error
		}
		LogsWithSigs *struct {
			InputStart int64
			InputEnd   int64
			OutputLogs []logpoller.Log
			OutputErr  error
		}
	}{
		{
			Name:        "LatestBlockError",
			ExpectedErr: ErrHeadNotAvailable,
			LatestBlock: &struct {
				OutputBlock int64
				OutputErr   error
			}{
				OutputBlock: 0,
				OutputErr:   fmt.Errorf("test error output"),
			},
		},
		{
			Name:             "LastHeadPollIsLatestHead",
			LastPoll:         500,
			ExpectedLastPoll: 500,
			ExpectedErr:      nil,
			LatestBlock: &struct {
				OutputBlock int64
				OutputErr   error
			}{
				OutputBlock: 500,
				OutputErr:   nil,
			},
		},
		{
			Name:             "LastHeadPollNotInitialized",
			LastPoll:         0,
			ExpectedLastPoll: 500,
			ExpectedErr:      nil,
			LatestBlock: &struct {
				OutputBlock int64
				OutputErr   error
			}{
				OutputBlock: 500,
				OutputErr:   nil,
			},
		},
		{
			Name:             "LogPollError",
			LastPoll:         480,
			Address:          common.BigToAddress(big.NewInt(1)),
			ExpectedLastPoll: 500,
			ExpectedErr:      ErrLogReadFailure,
			LatestBlock: &struct {
				OutputBlock int64
				OutputErr   error
			}{
				OutputBlock: 500,
				OutputErr:   nil,
			},
			LogsWithSigs: &struct {
				InputStart int64
				InputEnd   int64
				OutputLogs []logpoller.Log
				OutputErr  error
			}{
				InputStart: 250,
				InputEnd:   500,
				OutputLogs: []logpoller.Log{},
				OutputErr:  fmt.Errorf("test output error"),
			},
		},
		{
			Name:             "LogPollSuccess",
			LastPoll:         480,
			Address:          common.BigToAddress(big.NewInt(1)),
			ExpectedLastPoll: 500,
			ExpectedErr:      nil,
			LatestBlock: &struct {
				OutputBlock int64
				OutputErr   error
			}{
				OutputBlock: 500,
				OutputErr:   nil,
			},
			LogsWithSigs: &struct {
				InputStart int64
				InputEnd   int64
				OutputLogs []logpoller.Log
				OutputErr  error
			}{
				InputStart: 250,
				InputEnd:   500,
				OutputLogs: []logpoller.Log{
					{EvmChainId: utils.NewBig(big.NewInt(5)), LogIndex: 1},
					{EvmChainId: utils.NewBig(big.NewInt(6)), LogIndex: 2},
				},
				OutputErr: nil,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			mp := new(mocks.LogPoller)

			if test.LatestBlock != nil {
				mp.On("LatestBlock", mock.Anything).
					Return(test.LatestBlock.OutputBlock, test.LatestBlock.OutputErr)
			}

			if test.LogsWithSigs != nil {
				fc := test.LogsWithSigs
				mp.On("LogsWithSigs", fc.InputStart, fc.InputEnd, upkeepStateEvents, test.Address, mock.Anything).Return(fc.OutputLogs, fc.OutputErr)
			}

			rg := &EvmRegistry{
				addr:          test.Address,
				lastPollBlock: test.LastPoll,
				poller:        mp,
				chLog:         make(chan logpoller.Log, 10),
			}

			err := rg.pollLogs()

			assert.Equal(t, test.ExpectedLastPoll, rg.lastPollBlock)
			if test.ExpectedErr != nil {
				assert.ErrorIs(t, err, test.ExpectedErr)
			} else {
				assert.Nil(t, err)
			}

			var outputLogCount int

		CheckLoop:
			for {
				chT := time.NewTimer(20 * time.Millisecond)
				select {
				case l := <-rg.chLog:
					chT.Stop()
					if test.LogsWithSigs == nil {
						assert.FailNow(t, "logs detected but no logs were expected")
					}
					outputLogCount++
					assert.Contains(t, test.LogsWithSigs.OutputLogs, l)
				case <-chT.C:
					break CheckLoop
				}
			}

			if test.LogsWithSigs != nil {
				assert.Equal(t, len(test.LogsWithSigs.OutputLogs), outputLogCount)
			}

			mp.AssertExpectations(t)
		})
	}
}

func TestRegistry_GetBlockAndUpkeepId(t *testing.T) {
	r := &EvmRegistry{}
	tests := []struct {
		name       string
		input      ocr2keepers.UpkeepPayload
		wantBlock  *big.Int
		wantUpkeep *big.Int
	}{
		{
			"happy flow",
			ocr2keepers.UpkeepPayload{
				Upkeep: ocr2keepers.ConfiguredUpkeep{
					ID: ocr2keepers.UpkeepIdentifier([]byte("10")),
				},
				Trigger: ocr2keepers.Trigger{
					BlockNumber: 1,
					BlockHash:   common.Bytes2Hex([]byte{1, 2, 3, 4, 5, 6, 7, 8}),
				},
			},
			big.NewInt(1),
			big.NewInt(0).SetBytes([]byte("10")),
		},
		{
			"empty block number",
			ocr2keepers.UpkeepPayload{
				Upkeep: ocr2keepers.ConfiguredUpkeep{
					ID: ocr2keepers.UpkeepIdentifier([]byte("10")),
				},
			},
			big.NewInt(0),
			big.NewInt(0).SetBytes([]byte("10")),
		},
		{
			"empty payload",
			ocr2keepers.UpkeepPayload{},
			big.NewInt(0),
			big.NewInt(0),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			block, upkeep := r.getBlockAndUpkeepId(tc.input)
			assert.Equal(t, tc.wantBlock, block)
			assert.Equal(t, tc.wantUpkeep, upkeep)
		})
	}
}
