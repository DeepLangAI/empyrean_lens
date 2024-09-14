package conf

import "testing"

func Test_ensureDirExists(t *testing.T) {
	ensureDirExists("test/1/2/3")
}
