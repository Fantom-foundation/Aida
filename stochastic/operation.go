package stochastic

// IDs of StateDB Operations for event collection
const (
	addBalanceID = iota
	createAccountID
	emptyID
	existID
	finaliseID
	getBalanceID
	getCodeHashID
	getCodeID
	getCodeSizeID
	getCommittedStateID
	getNonceID
	getStateID
	hasSuicidedID
	revertToSnapshotID
	setCodeID
	setNonceID
	setStateID
	snapshotID
	subBalanceID
	suicideID

	numOps
)

// Abbreviated Operation Label for concise rendering
var opText = []string{
	addBalanceID:        "AB",
	createAccountID:     "CA",
	emptyID:             "EM",
	existID:             "EX",
	finaliseID:          "FI",
	getBalanceID:        "GB",
	getCodeHashID:       "GH",
	getCodeID:           "GC",
	getCodeSizeID:       "GZ",
	getCommittedStateID: "GM",
	getNonceID:          "GN",
	getStateID:          "GS",
	hasSuicidedID:       "HS",
	revertToSnapshotID:  "RS",
	setCodeID:           "SC",
	setNonceID:          "SN",
	snapshotID:          "SN",
	subBalanceID:        "SB",
	setStateID:          "SS",
	suicideID:           "SU",
}
