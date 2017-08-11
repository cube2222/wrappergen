package examples

import "context"

type ExampleInterface interface {
	ExampleFunctionNoError(context.Context, int) int
	ExampleFunctionWithError(context.Context, int) (string, error)
}
