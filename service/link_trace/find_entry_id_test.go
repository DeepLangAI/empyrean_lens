package link_trace

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_extractEntryId(t *testing.T) {
	entryId := extractEntryId("summary start, file_id:, url_id:673ff45948fa8f05c1dd5915 generate_type:1.", "url_id:")
	fmt.Println(entryId)
	assert.True(t, entryId == "673ff45948fa8f05c1dd5915")
}
