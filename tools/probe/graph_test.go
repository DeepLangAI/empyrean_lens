package probe

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"testing"
)

func TestInitGraph(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	empyrean_lens.Init(ctx)
	functionMap := RegisterFunctions()

	graph, err := LoadGraphFromConfig("/Users/wh/Documents/DeepLang/empyrean_lens/conf/graph.json", functionMap)
	if err != nil {
		log.Fatalf("Failed to load graph from config: %v", err)
	}

	graph.PrintGraph()
	graph.Trace(ctx)
}

func TestEditJson(t *testing.T) {
	s := `
{
  "edges": [
    {"from": "0", "to": "1"},
    {"from": "1", "to": "3"},
    {"from": "1", "to": "4"},
    {"from": "4", "to": "8"},
    {"from": "8", "to": "5"},
    {"from": "8", "to": "6"},
    {"from": "8", "to": "7"},
    {"from": "0", "to": "2"},
    {"from": "3", "to": "9"},
    {"from": "9", "to": "10"},
    {"from": "9", "to": "11"},
    {"from": "9", "to": "12"}
  ],
  "functions": {
    "0": {"f_name": "Root", "label": "root", "status": 1},
    "1": {"f_name": "Login", "label": "login"},
    "2": {"f_name": "Logout", "label": "logout"},
    "3": {"f_name": "UploadPDF", "label": "上传PDF"},
    // "4": {"f_name": "UploadURL", "label": "上传URL"},
    // "5": {"f_name": "UrlAbstract", "label": "生成概述"},
    "6": {"f_name": "UrlViewpoint", "label": "生成关键信息"},
    "7": {"f_name": "UrlOutline", "label": "生成大纲"},
    "8": {"f_name": "UrlDldParse", "label": "URL解析"},
    "9": {"f_name": "PdfParse", "label": "PDF解析"},
    "10": {"f_name": "PdfAbstract", "label": "生成概述"},
    "11": {"f_name": "PdfViewpoint", "label": "生成关键信息"},
    "12": {"f_name": "PdfOutline", "label": "生成大纲"}
  }
}
`
	// 去除所有行中，以#开始的内容，使用正则
	re := regexp.MustCompile(`//.*(\n|$)`)
	print(re.ReplaceAllString(s, ""))
}

func Test_registerFunctions(t *testing.T) {
	ctx := context.Background()
	funcs := RegisterFunctions()
	for fname, fun := range funcs {
		fmt.Printf("Function name: %s, Function: %v\n", fname, fun(ctx))
	}
}

func Test_RegisterMethods(t *testing.T) {
	ctx := context.Background()
	a := NewAgent()
	funcMap := map[string]func(context.Context) bool{}
	objValue := reflect.ValueOf(a)
	objType := reflect.TypeOf(a)
	for i := 0; i < objValue.NumMethod(); i++ {
		method := objValue.Method(i)
		methodName := objType.Method(i).Name
		funcMap[methodName] = method.Interface().(func(ctx context.Context) bool)
	}
	for fname, fun := range funcMap {
		fmt.Printf("Function name: %s, Function: %v\n", fname, fun(ctx))
	}
}
