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

package logger

import (
	"strings"

	"github.com/Fantom-foundation/Aida/ethtest"
	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/txcontext"
	"github.com/Fantom-foundation/Aida/utils"
)

type ethStateTestLogger struct {
	extension.NilExtension[txcontext.TxContext]
	cfg                 *utils.Config
	log                 logger.Logger
	previousTestLabel   string
	overall             int
	currentLabelCounter map[string]int
}

func MakeEthStateTestLogger(cfg *utils.Config) executor.Extension[txcontext.TxContext] {
	return makeEthStateTestLogger(cfg, logger.NewLogger(cfg.LogLevel, "EthStateTestLogger"))
}

func makeEthStateTestLogger(cfg *utils.Config, log logger.Logger) executor.Extension[txcontext.TxContext] {
	return &ethStateTestLogger{
		cfg:                 cfg,
		log:                 log,
		overall:             0,
		currentLabelCounter: make(map[string]int),
	}
}

// PreTransaction reports test name and fork.
func (l *ethStateTestLogger) PreTransaction(s executor.State[txcontext.TxContext], _ *executor.Context) error {
	// cast state.Data to stJSON
	c := s.Data.(*ethtest.StJSON)

	// Print only new version of test
	if strings.Compare(l.previousTestLabel, c.TestLabel) != 0 {
		l.log.Noticef("Currently iterating %v", c.TestLabel)
		l.previousTestLabel = c.TestLabel
		l.currentLabelCounter = make(map[string]int)
	}
	l.currentLabelCounter[c.UsedNetwork]++
	l.log.Infof(" Running test fork: %v number %v; ", c.UsedNetwork, l.currentLabelCounter[c.UsedNetwork])
	l.overall++
	return nil
}

// PostRun reports total tests run.
func (l *ethStateTestLogger) PostRun(executor.State[txcontext.TxContext], *executor.Context, error) error {
	l.log.Noticef("Total %v tests processed.", l.overall)
	return nil
}
