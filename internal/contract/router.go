package contract

import (
	"fmt"
	"reflect"
	"strings"
)

// Router is responsible for finding and calling functions using Reflection
type Router struct{}

// CallMethod
func (r *Router) CallMethod(app interface{}, sender, methodName string, args []string) (any, error, error) {
	// 1. Get information about the App object (Reflection)
	val := reflect.ValueOf(app)

	// 2. Find function by name
	// Note: The function must be capitalized (Public) to be found
	method := val.MethodByName(methodName)
	// check sender

	if !method.IsValid() {
		// Try to find the capitalized function (if the user accidentally sent lowercase)
		method = val.MethodByName(strings.Title(methodName))
		if !method.IsValid() {
			return nil, fmt.Errorf("function '%s' does not exist in the contract", methodName), nil
		}
	}

	// 3. Prepare parameters for the function call
	// Convention: User's function must accept (a, b, c)
	methodType := method.Type()
	numIn := methodType.NumIn()

	if len(args) != numIn {
		return nil, fmt.Errorf("expected %d arguments, got %d", numIn, len(args)), nil
	}

	var inputArgs []reflect.Value
	for i := 0; i < numIn; i++ {
		paramType := methodType.In(i)
		argVal := reflect.ValueOf(args[i])

		// Neu tham so khac kieu string (vi du int), co the xu ly convert o day
		if argVal.Type() != paramType {
			argVal = argVal.Convert(paramType)
		}
		inputArgs = append(inputArgs, argVal)
	}

	// 4. CALL FUNCTION (Invoke)
	results := method.Call(inputArgs)

	// 5. Process the return results
	var resData any
	var err error

	if len(results) == 1 {
		if results[0].IsValid() && !results[0].IsZero() {
			if e, ok := results[0].Interface().(error); ok {
				err = e
			} else {
				resData = results[0].Interface()
			}
		}
	} else if len(results) >= 2 {
		if results[0].IsValid() {
			resData = results[0].Interface()
		}

		if results[1].IsValid() && !results[1].IsZero() {
			if e, ok := results[1].Interface().(error); ok {
				err = e
			}
		}
	}

	return resData, err, nil
}
