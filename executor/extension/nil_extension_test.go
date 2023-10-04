package extension

import (
	"testing"

	"github.com/Fantom-foundation/Aida/executor"
)

func TestNilExtensionIsExtension(t *testing.T) {
	var _ executor.Extension[any] = NilExtension[any]{}
}
