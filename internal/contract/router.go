package contract

import (
	"encoding/json"
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
	// Convention: User's function must accept (a, b, c)
	methodType := method.Type()
	numIn := methodType.NumIn()

	if len(args) != numIn {
		return nil, fmt.Errorf("expected %d arguments, got %d", numIn, len(args))
	}

	var inputArgs []reflect.Value
	for i := 0; i < numIn; i++ {
		paramType := methodType.In(i)
		argVal := reflect.ValueOf(string(args[i]))

		// Neu tham so khac kieu string (vi du int), co the xu ly convert o day
		if argVal.Type() != paramType {
			argVal = argVal.Convert(paramType)
		}
		inputArgs = append(inputArgs, argVal)
	}

	// 4. CALL FUNCTION (Invoke)
	results := method.Call(inputArgs)

	// 5. Process the return results
	if len(results) == 1 {
		if !results[0].IsNil() {
			if err, ok := results[0].Interface().(error); ok {
				return nil, err
			}
		}
		return nil, nil
	} else if len(results) >= 2 {
		var resBytes []byte
		if !results[0].IsNil() {
			switch v := results[0].Interface().(type) {
			case []byte:
				resBytes = v
			case string:
				resBytes = []byte(v)
			case []string:
				resBytes, _ = json.Marshal(v)
			}
		}

		var err error
		if !results[1].IsNil() {
			if e, ok := results[1].Interface().(error); ok {
				err = e
			}
		}
		return resBytes, err
	}

	return nil, nil
}
