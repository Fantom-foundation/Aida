package stochastic

// Operation IDs of event collection
const (
	// operations with no simulation arguments
	StochasticFinaliseID = iota
	StochasticSnapshotID
	StochasticRevertToSnapshotID

	// Operations with a contract address as an argument.
	StochasticAddBalanceID
	StochasticCreateAccountID
	StochasticEmptyID
	StochasticExistID
	StochasticGetBalanceID
	StochasticGetCodeHashID
	StochasticGetCodeID
	StochasticGetCodeSizeID
	StochasticGetNonceID
	StochasticHasSuicidedID
	StochasticSetCodeID
	StochasticSetNonceID
	StochasticSubBalanceID
	StochasticSuicideID

	// Operations with a contract address and storage key.
	StochasticGetCommittedStateID
	StochasticGetStateID

	// Operations with a contract address, storage key & value.
	StochasticSetStateID

	numStochasticOps
)

// Operation text
// NB: order must follow order as defined above.
var operationText = []string{
	"Finalise",
	"SnapshotID",
	"RevertToSnapshotID",
	"AddBalanceID",
	"CreateAccountID",
	"EmptyID",
	"ExistID",
	"GetBalanceID",
	"GetCodeHashID",
	"GetCodeID",
	"GetCodeSizeID",
	"GetNonceID",
	"HasSuicidedID",
	"SetCodeID",
	"SetNonceID",
	"SubBalanceID",
	"SuicideID",
	"GetCommittedStateID",
	"GetStateID",
	"SetStateID",
}

// Operation text
// NB: order must follow order as defined above.
var operationNumArgs = []int{
	0,
	0,
	0,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	1,
	2,
	2,
	3,
}
