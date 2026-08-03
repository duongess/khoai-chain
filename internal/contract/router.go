package contract

import (
	"fmt"
	"reflect"
	"strings"
)

// Router is responsible for finding and calling functions using Reflection
type Router struct{}

// CallMethod
func (r *Router) CallMethod(app interface{}, sender, methodName []byte, args [][]byte) ([]byte, error) {
	// 1. Get information about the App object (Reflection)
	val := reflect.ValueOf(app)

	// 2. Find function by name
	// Note: The function must be capitalized (Public) to be found
	method := val.MethodByName(string(methodName))
	// check sender

	if !method.IsValid() {
		// Try to find the capitalized function (if the user accidentally sent lowercase)
		method = val.MethodByName(strings.Title(string(methodName)))
		if !method.IsValid() {
			return nil, fmt.Errorf("function '%s' does not exist in the contract", methodName)
		}
	}

	// 3. Prepare parameters for the function call
	// Convention: User's function must accept ([][]byte)
	inputArgs := []reflect.Value{reflect.ValueOf(args)}

	// 4. CALL FUNCTION (Invoke)
	results := method.Call(inputArgs)

	// 5. Process the return results
	// Convention: User's function must return ([]byte, error)
	if len(results) < 2 {
		return nil, fmt.Errorf("function must return 2 values: ([]byte, error)")
	}

	// Get the result (Bytes)
	resBytes := results[0].Interface().([]byte)

	// Get the error (Error)
	errObj := results[1].Interface()
	var err error
	if errObj != nil {
		err = errObj.(error)
	}

	return resBytes, err
}
