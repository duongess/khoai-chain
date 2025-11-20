package examples

import (
	"khoai-chain/internal/contract"
)

type UsageExamples struct {
	contract.BaseContract
}

func NewUsageExamples() *UsageExamples {
	app := &UsageExamples{}
	app.SetName("examplesgolang")
	return app
}
