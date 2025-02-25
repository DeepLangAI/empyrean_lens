package gse

import (
	"fmt"
	"testing"
)

func TestCutTextV1_EmptyInput(t *testing.T) {

	query := "6762b26e1c8cda9b178a2718"
	tokens := InitGse().CutTextV1(query)
	fmt.Println(tokens)
}
