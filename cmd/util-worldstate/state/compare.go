// Package state implements executable entry points to the world state generator app.
package state

//
//// CmdCompareState compares states of two databases whether they are identical
//var CmdCompareState = cli.Command{
//	Action:      compareDb,
//	Name:        "compare",
//	Aliases:     []string{"cmp"},
//	Usage:       "Compare whether states of two databases are identical.",
//	Description: `Compares given snapshot database against target snapshot database.`,
//	ArgsUsage:   "<to>",
//	Flags: []cli.Flag{
//		&utils.TargetDbFlag,
//	},
//}
//
//// compareDb compares world state stored inside source and destination databases.
//func compareDb(ctx *cli.Context) error {
//	// make config
//	cfg, err := utils.NewConfig(ctx, utils.LastBlockArg)
//	if err != nil {
//		return err
//	}
//
//	// try to open state DB
//	stateDB, err := snapshot.OpenStateDB(cfg.WorldStateDb)
//	if err != nil {
//		return err
//	}
//	defer snapshot.MustCloseStateDB(stateDB)
//
//	path, err := DefaultPath(ctx, &utils.TargetDbFlag, "clone")
//	if err != nil {
//		return err
//	}
//
//	// try to open target state DB
//	stateRefDB, err := snapshot.OpenStateDB(path)
//	if err != nil {
//		return err
//	}
//	defer snapshot.MustCloseStateDB(stateRefDB)
//
//	// make logger
//	log := logger.NewLogger(cfg.LogLevel, "cmp")
//	log.Infof("comparing %s against %s", cfg.WorldStateDb, cfg.TargetDb)
//
//	// call CompareTo against target database
//	err = stateDB.CompareTo(context.Background(), stateRefDB)
//	if err != nil {
//		return fmt.Errorf("compare failed; %s", err.Error())
//	}
//
//	log.Info("databases are identical")
//	return nil
//}
