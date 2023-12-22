package proxy

import (
	"fmt"
	"sort"
	"strings"

	substate "github.com/Fantom-foundation/Substate"
	"github.com/ethereum/go-ethereum/common"
)

// PrettySubstateAlloc is a wrapper over a SubstateAlloc that adds human
// readable, stable --and thus diff-able -- pretty printing to it.
type PrettySubstateAlloc substate.SubstateAlloc

func (a PrettySubstateAlloc) String() string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("SubstateAlloc{\n\tsize: %d\n", len(a)))
	keys := []common.Address{}
	for key := range a {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })

	builder.WriteString("\tAccounts:\n")
	for _, key := range keys {
		builder.WriteString(fmt.Sprintf("\t\t%x: %v\n", key, prettySubstateAccount{a[key]}))
	}
	builder.WriteString("}")
	return builder.String()
}

type prettySubstateAccount struct {
	*substate.SubstateAccount
}

func (a prettySubstateAccount) String() string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("Account{\n\t\t\tnonce: %d\n\t\t\tbalance %v\n", a.Nonce, a.Balance))

	builder.WriteString("\t\t\tStorage{\n")
	keys := []common.Hash{}
	for key := range a.Storage {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })
	for _, key := range keys {
		builder.WriteString(fmt.Sprintf("\t\t\t\t%v=%v\n", key, a.Storage[key]))
	}
	builder.WriteString("\t\t\t}\n\t\t}")
	return builder.String()
}
