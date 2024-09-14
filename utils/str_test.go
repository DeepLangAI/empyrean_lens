package utils

import "testing"

func TestJSONMarshal(t *testing.T) {
	p := Person{
		Name: "John",
		Age:  30,
	}
	marshal := JSONMarshal(p)
	t.Log(marshal)
}
