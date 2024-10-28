package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestJSONMarshal(t *testing.T) {
	p := Person{
		Name: "John",
		Age:  30,
	}
	marshal := JSONMarshal(p)
	t.Log(marshal)
}

func TestDecodeMIME(t *testing.T) {
	s := "=?utf-8?b?5YmN5pa55qih5Z6L5Y2H57qn77yM6K+36YeN5paw5LiK5Lyg5paH56ug6YeN6K+V772e?="
	assert.True(t, DecodeMIME(s) == "前方模型升级，请重新上传文章重试～")
}

func TestContains(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		assert.True(t, Contains([]int{1, 2, 3}, 2))
	})
	t.Run("str", func(t *testing.T) {
		assert.True(t, Contains([]string{"a", "b", "c"}, "b"))
	})
}
