package main

import (
	"strings"
	"regexp"
	"fmt"
	"strconv"
)

type AdderRuntime struct {
	Functions []*RuntimeFunction
	Listeners []*RuntimeListener
}

type RuntimeFunction struct {
	ReturnType VariableType
	Name       string
	Parameters []FunctionParameter
	InternalId int
}

type RuntimeListener struct {
	Name string
	Parameters []FunctionParameter
	InternalId int
}

type FunctionParameter struct {
	Type VariableType
	Name string
}

var AnyType = "void|int|string|bool"
var RuntimeLinePattern, _ = regexp.Compile("^\\s*(" + AnyType + "|listener)\\s+([a-zA-Z_][a-zA-Z0-9_]*)\\((.*)\\)\\s*->\\s*(\\d+)\\s*;$")
var ParametersPattern, _ = regexp.Compile("\\s*(" + AnyType + ")\\s+([a-zA-Z_0-9]+)")

func ParseRuntime(runtimeData string) (*AdderRuntime, error) {
	runtime := AdderRuntime{}

	lines := strings.Split(runtimeData, "\n")
	for lineNumber, line := range lines {
		line = strings.TrimSpace(line)

		// Skip lines that are empty, comment or start with //
		if len(line) < 1 || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		matches := RuntimeLinePattern.FindAllStringSubmatch(line, -1)
		if len(matches) == 1 && len(matches[0]) >= 5 {
			groups := matches[0][1:]

			returnType := groups[0]
			methodName := groups[1]
			parameters := groups[2]
			uid := groups[3]

			uidInt, err := strconv.Atoi(uid)
			if err != nil {
				return nil, fmt.Errorf("cannot convert uid into number: %s (%s)", uid, err)
			}

			// Parse the parameters into parameter structs..
			parsedParameters, err := parseParameters(parameters)
			if err != nil {
				return nil, fmt.Errorf("error parsing parameters at line %d: %s", lineNumber+1, err)
			}

			if returnType == "listener" {
				// Put the new listener into the list of listeners.
				listener := &RuntimeListener{
					Name:       methodName,
					Parameters: parsedParameters,
					InternalId: uidInt,
				}

				runtime.Listeners = append(runtime.Listeners, listener)
			} else {
				// Parse return type
				resolvedType := ResolveType(returnType)
				if resolvedType == VarTypeUnresolved {
					return nil, fmt.Errorf("invalid return type for method at line %d: %s", lineNumber+1, returnType)
				}

				// Create a new function and continue to the next line.
				function := &RuntimeFunction{
					ReturnType: resolvedType,
					Name:       methodName,
					Parameters: parsedParameters,
					InternalId: uidInt,
				}

				runtime.Functions = append(runtime.Functions, function)
			}
		} else {
			return nil, fmt.Errorf("invalid runtime definition at line %d", lineNumber+1)
		}
	}

	return &runtime, nil
}

// parseParameters parses a single parameters string into an array of parameters.
func parseParameters(parameters string) ([]FunctionParameter, error) {
	var result []FunctionParameter
	if len(strings.TrimSpace(parameters)) < 1 {
		return result, nil
	}

	parametersSplit := strings.Split(parameters, ",") // Whitespace is dealt with in the regexp

	for parameterNumber, param := range parametersSplit {
		matches := ParametersPattern.FindAllStringSubmatch(param, -1)

		if len(matches) > 0 {
			paramType := matches[0][1]
			paramName := matches[0][2]

			// Resolve the type to a variable type that the compiler can deal with
			resolvedType := ResolveVarType(paramType)
			if resolvedType == VarTypeUnresolved {
				return nil, fmt.Errorf("invalid variable type for parameter %d ('%s'): %s", parameterNumber+1, paramName, paramType)
			}

			param := FunctionParameter{Name: paramName, Type: resolvedType}
			result = append(result, param)
		} else {
			return nil, fmt.Errorf("could not parse parameter %d", parameterNumber+1)
		}
	}

	return result, nil
}

func (r *AdderRuntime) FindFunction(name string) *RuntimeFunction {
	for _, v := range r.Functions {
		if v.Name == name {
			return v
		}
	}

	return nil
}

func (r *AdderRuntime) FindFunctionById(uid int) *RuntimeFunction {
	for _, v := range r.Functions {
		if v.InternalId == uid {
			return v
		}
	}

	return nil
}

func (r *AdderRuntime) FindListener(name string) *RuntimeListener {
	for _, v := range r.Listeners {
		if v.Name == name {
			return v
		}
	}

	return nil
}