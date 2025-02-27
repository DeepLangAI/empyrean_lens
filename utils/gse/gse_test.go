package gse

import (
	"fmt"
	"testing"
)

func TestCutTextV1_EmptyInput(t *testing.T) {

	query := "6711ff7cb9b31534c35511fc"
	tokens := InitGse().CutTextV1(query)
	fmt.Println(tokens)
}
