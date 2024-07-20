package probe

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/dal/mongo"
	"fmt"
	"log"
	"reflect"
	"testing"
)

func TestInitGraph(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	mongo.Init(ctx)
	functionMap := RegisterFunctions()

	graph, err := LoadGraphFromConfig("/Users/wh/Documents/DeepLang/empyrean_lens/conf/graph.json", functionMap)
	if err != nil {
		log.Fatalf("Failed to load graph from config: %v", err)
	}

	graph.PrintGraph()
	graph.Trace(ctx)
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
