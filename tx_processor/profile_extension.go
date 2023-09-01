package tx_processor

import (
	"fmt"

	"github.com/Fantom-foundation/Aida/utils"
)

// ProfileExtension provide the logging action for tx processing
type ProfileExtension struct {
	ProcessorExtensions
}

// NewProfileExtension creates a new logging action for tx processing.
func NewProfileExtension() *ProfileExtension {
	return &ProfileExtension{}
}

// Init opens the CPU profiler if specied in the cli.
func (ext *ProfileExtension) Init(bp *TxProcessor) error {
	// CPU profiling (if enabled)
	if err := utils.StartCPUProfile(bp.cfg); err != nil {
		return fmt.Errorf("failed to open CPU profiler; %v", err)
	}
	return nil
}

func (ext *ProfileExtension) PostPrepare(bp *TxProcessor) error {
	return nil
}

func (ext *ProfileExtension) PostBlock(bp *TxProcessor) error {
	return nil
}

func (ext *ProfileExtension) PostTransaction(bp *TxProcessor) error {
	return nil
}

// PostProcessing issues a memory profile report.
func (ext *ProfileExtension) PostProcessing(bp *TxProcessor) error {
	// write memory profile
	if err := utils.StartMemoryProfile(bp.cfg); err != nil {
		return err
	}
	return nil
}

// Exit stops CPU profiling.
func (ext *ProfileExtension) Exit(bp *TxProcessor) error {
	utils.StopCPUProfile(bp.cfg)
	return nil
}
