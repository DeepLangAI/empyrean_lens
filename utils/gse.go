package utils

import (
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/go-ego/gse"
)

var (
	myGse   *Gse
	onceGse sync.Once
)

type Gse struct {
	segmenter gse.Segmenter
}

func InitGse() *Gse {
	onceGse.Do(func() {
		segmenter, _ := gse.New("zh,testdata/test_en_dict3.txt", "alpha")
		segmenter.LoadDict()
		myGse = &Gse{
			segmenter: segmenter,
		}
	})
	return myGse
}

func (g *Gse) CutTextV1(text string) []string {
	return g.segmenter.Cut(text, true)
}

func (g *Gse) CutTextV2(text string) []string {
	var filteredText strings.Builder
	for _, char := range text {
		if !unicode.IsPunct(char) {
			filteredText.WriteRune(char)
		} else {
			filteredText.WriteString(" ")
		}
	}
	text = strings.Replace(filteredText.String(), "\n", " ", -1)
	reg := regexp.MustCompile(`( )+|(\n)+`)
	text = reg.ReplaceAllString(text, "$1$2")
	return strings.Split(text, " ")
}
