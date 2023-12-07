package dbcomponent

import (
	"fmt"
)

type DbComponent string

const (
	All       DbComponent = "all"
	Substate  DbComponent = "substate"
	Delete    DbComponent = "delete"
	Update    DbComponent = "update"
	StateHash DbComponent = "state-hash"
)

// ParseDbComponent parses string to DbComponent
func ParseDbComponent(s string) (DbComponent, error) {
	switch s {
	case "all":
		return All, nil
	case "substate":
		return Substate, nil
	case "delete":
		return Delete, nil
	case "update":
		return Update, nil
	case "state-hash":
		return StateHash, nil
	default:
		return "", fmt.Errorf("invalid db component: %v. Usage: (\"all\", \"substate\", \"delete\", \"update\", \"state-hash\")", s)
	}
}
