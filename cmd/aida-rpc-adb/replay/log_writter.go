package replay

import (
	"bufio"
	"fmt"
	"os"

	"github.com/Fantom-foundation/Aida/executor"
	"github.com/Fantom-foundation/Aida/executor/extension"
	"github.com/Fantom-foundation/Aida/logger"
	"github.com/Fantom-foundation/Aida/rpc_iterator"
	"github.com/Fantom-foundation/Aida/utils"
)

func makeResultComparator(cfg *utils.Config, input chan *comparatorError) executor.Extension[*rpc_iterator.RequestWithResponse] {
	return &resultComparator{
		cfg:   cfg,
		log:   logger.NewLogger(cfg.LogLevel, "rpc-adb-comparator"),
		input: input,
	}
}

type resultComparator struct {
	extension.NilExtension[*rpc_iterator.RequestWithResponse]
	cfg    *utils.Config
	log    logger.Logger
	input  chan *comparatorError
	writer *bufio.Writer
	file   *os.File
}

func (c *resultComparator) PreRun(executor.State[*rpc_iterator.RequestWithResponse], *executor.Context) error {
	var err error
	c.file, err = os.Create(c.cfg.LogFile)
	if err != nil {
		return fmt.Errorf("cannot create log file; %v", err)
	}

	c.writer = bufio.NewWriter(c.file)
	go c.doWrite()

	return nil
}

func (c *resultComparator) PostRun(executor.State[*rpc_iterator.RequestWithResponse], *executor.Context, error) error {
	return c.file.Close()
}

func (c *resultComparator) doWrite() {
	for {
		in := <-c.input

		_, err := c.writer.WriteString(in.Error())
		if err != nil {
			c.log.Errorf("cannot write; %v", err)
		}

		err = c.writer.Flush()
		if err != nil {
			c.log.Errorf("cannot flush writer; %v", err)
		}
	}
}
