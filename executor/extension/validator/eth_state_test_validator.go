// Copyright 2024 Fantom Foundation
// This file is part of Aida Testing Infrastructure for Sonic
//
// Aida is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Aida is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Aida. If not, see <http://www.gnu.org/licenses/>.

package validator

import (
	"errors"
	"fmt"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"
)

func MakeEthStateTestValidator(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	if !cfg.Validate {
		return extension.NilExtension[txcontext.TxContext]{}
	}
	return makeEthStateTestValidator(cfg, logger.NewLogger(cfg.LogLevel, "EthStateTestValidator"))
}

func makeEthStateTestValidator(cfg *utils.Config, log logger.Logger) executor.Extension[txcontext.TxContext] {
	return &ethStateTestValidator{
		cfg: cfg,
		log: log,
	}
}

type ethStateTestValidator struct {
	extension.NilExtension[txcontext.TxContext]
	cfg            *utils.Config
	log            logger.Logger
	numberOfErrors int
	finalErr       error
}

func (e *ethStateTestValidator) PreTransaction(s executor.State[txcontext.TxContext], ctx *executor.Context) error {
	err := validateWorldState(e.cfg, ctx.State, s.Data.GetInputState(), e.log)
	if err != nil {
		return fmt.Errorf("pre alloc validation failed; %v", err)
	}

	return nil
}

func (e *ethStateTestValidator) PostTransaction(state executor.State[txcontext.TxContext], ctx *executor.Context) error {
	// todo add total counter

	// todo add log rlpHash validation in separate PR
	//e.log.Notice(rlpHash(ctx.State.GetLogs(common.Hash{}, 0, common.Hash{})))

	var err error
	_, got := ctx.ExecutionResult.GetRawResult()
	_, want := state.Data.GetResult().GetRawResult()
	if want == nil && got == nil {
		return nil
	}
	if got == nil && want != nil {
		// todo temporal fix - this gets checked before apply message in go-ethereum-sonic/tests/state_test_util.go 256
		if len(state.Data.GetMessage().BlobHashes)*params.BlobTxBlobGasPerBlob > params.MaxBlobGasPerBlock {
			return nil
		}
		err = fmt.Errorf("expected error %w, got no error", want)
	}
	if got != nil && want == nil {
		err = fmt.Errorf("unexpected error: %w", got)
	}
	if want != nil && got != nil {
		// TODO check error string - requires somewhat complex string parsing
		return nil
	}

	if !e.cfg.ContinueOnFailure {
		return err
	}

	ctx.ErrorInput <- err
	e.finalErr = errors.Join(e.finalErr, fmt.Errorf("ERROR VALIDATION FAILED: %s\n\tERR: \n\t%w", state.Data, err))
	e.numberOfErrors++

	// endless run
	if e.cfg.MaxNumErrors == 0 {
		return nil
	}

	// too many errors
	if e.numberOfErrors >= e.cfg.MaxNumErrors {
		return e.finalErr
	}

	return nil
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewLegacyKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}
