package blockprocessor

import (
	"fmt"
	"reflect"
)

// ProcessorExtensions supports block processing actions
// NOTE: ALl methods MUST be implemented.
// If you do not find any functionality for a method, simply make it return nil.
type ProcessorExtensions interface {
	Init(*BlockProcessor) error        // Initialise action (before block processing starts)
	PostPrepare(*BlockProcessor) error // Post-prepare action (after statedb has been created/primed)
	PostBlock(*BlockProcessor) error
	PostTransaction(*BlockProcessor) error // Post-transaction action (after a transaction has been processed)
	PostProcessing(*BlockProcessor) error  // Post-processing action (after all transactions have been processed/before closing statedb)
	Exit(*BlockProcessor) error            // Exit action (after completing block processing)
}

// ExtensionFuncMap maps func name to func callback
type ExtensionFuncMap map[string]func(*BlockProcessor) error

// ExtensionList maps Extension name to Extension object
type ExtensionList map[string]ExtensionFuncMap

// NewExtensionList creates maps for all ProcessorExtensions function name to function callback for each method
func NewExtensionList(extensions []ProcessorExtensions) ExtensionList {
	l := make(ExtensionList)

	t := reflect.TypeOf((*ProcessorExtensions)(nil)).Elem()

	// BasicIterator over all chosen extensions
	for _, e := range extensions {
		m := make(ExtensionFuncMap)
		// BasicIterator over all methods inside extensions
		for i := 0; i < t.NumMethod(); i++ {
			// assign method name to callback
			m[t.Method(i).Name] = reflect.ValueOf(e).MethodByName(t.Method(i).Name).Interface().(func(*BlockProcessor) error)
		}

		// assign extension name to list of methods
		l[reflect.TypeOf(e).Elem().Name()] = m
	}

	return l
}

// executeExtensions executes a matching method name of actions in the action list.
func (al ExtensionList) executeExtensions(method string, bp *BlockProcessor) error {
	var err error

	bp.Log.Debug("Executing...")
	bp.Log.Debugf("Method: %v", method)

	for extensionName, action := range al {
		bp.Log.Debugf("Extension: %v", extensionName)
		if err = action[method](bp); err != nil {
			return fmt.Errorf("cannot call ExecuteExtension: %v; func name: %v; err: %v", extensionName, method, err)
		}

	}
	return nil
}
