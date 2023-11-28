package dbcompoment

import "errors"

type DbComponent string

const (
	All       DbComponent = "all"
	Substate  DbComponent = "substate"
	Delete    DbComponent = "delete"
	Update    DbComponent = "update"
	StateHash DbComponent = "state-hash"
)

func (d *DbComponent) Set(value string) error {
	switch value {
	case string(All), string(Substate), string(Delete), string(Update), string(StateHash):
		*d = DbComponent(value)
	default:
		return errors.New("invalid db component. Valid options are 'all', 'substate', 'delete', 'update', 'state-hash'")
	}
	return nil
}

func (d *DbComponent) String() string {
	return string(*d)
}
