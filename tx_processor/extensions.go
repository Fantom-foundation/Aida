package tx_processor

import (
	"fmt"
	"reflect"
)

// ProcessorExtensions supports block processing actions
// NOTE: ALl methods MUST be implemented.
// If you do not find any functionality for a method, simply make it return nil.
type ProcessorExtensions interface {
	Init(*TxProcessor) error           // Initialise action (before block processing starts)
	PostPrepare(*TxProcessor) error    // Post-prepare action (after statedb has been created/primed)
	PostProcessing(*TxProcessor) error // Post-processing action (after all transactions have been processed/before closing statedb)
	Exit(*TxProcessor) error           // Exit action (after completing block processing)
}

// ExtensionFuncMap maps func name to func callback
type ExtensionFuncMap map[string]func(*TxProcessor) error

// ExtensionList maps Extension name to Extension object
type ExtensionList map[string]ExtensionFuncMap

// NewExtensionList creates maps for all ProcessorExtensions function name to function callback for each method
func NewExtensionList(extensions []ProcessorExtensions) ExtensionList {
	l := make(ExtensionList, len(extensions))

	t := reflect.TypeOf((*ProcessorExtensions)(nil)).Elem()

	// iterate over all chosen extensions
	for _, e := range extensions {
		m := make(ExtensionFuncMap)
		// iterate over all methods inside extensions
		for i := 0; i < t.NumMethod(); i++ {
			// assign method name to callback
			m[t.Method(i).Name] = reflect.ValueOf(e).MethodByName(t.Method(i).Name).Interface().(func(*TxProcessor) error)
		}

		// assign extension name to list of methods
		l[reflect.TypeOf(e).Elem().Name()] = m
	}

	return l
}

// ExecuteExtensions executes a matching method name of actions in the action list.
func (al ExtensionList) ExecuteExtensions(method string, bp *TxProcessor) error {
	var err error

	bp.log.Debug("Executing...")
	bp.log.Debugf("Method: %v", method)

	for extensionName, action := range al {
		bp.log.Debugf("Extension: %v", extensionName)
		if err = action[method](bp); err != nil {
			return fmt.Errorf("cannot call ExecuteExtension: %v; func name: %v; err: %v", extensionName, method, err)
		}

	}
	return nil
}
