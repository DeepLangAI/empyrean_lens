package probe

import (
	"context"
	"empyrean_lens/dal/mongo"
	"encoding/json"
	"fmt"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io"
	"os"
	"reflect"
	"time"
)

type Action func(ctx context.Context) bool

// Node represents a node in the graph
type Node struct {
	Id       string
	Label    string
	Adjacent []*Node
	Function Action
	Status   int
}

const StatusExecute = 0
const StatusDummy = 1

// Graph represents the entire graph
type Graph struct {
	Nodes map[string]*Node // id to node
}

func trace(ctx context.Context, node *Node, reports *[]mongo.NodeDetail) {
	for _, adjacent := range node.Adjacent {
		hlog.CtxInfof(ctx, "[TRACE] Tracing from `%s` to `%s`", node.Label, adjacent.Label)
		if adjacent.Status == StatusExecute {
			report := mongo.NodeDetail{}
			report.Name = adjacent.Label
			startTime := time.Now()
			res := adjacent.Function(ctx)
			report.Cost = time.Since(startTime).Seconds()
			if res {
				report.Result = mongo.RESULT_SUCCESS
				hlog.CtxInfof(ctx, "[TRACE] Tracing from `%s` to `%s success", node.Label, adjacent.Label)
				trace(ctx, adjacent, reports)
			} else {
				hlog.CtxInfof(ctx, "[TRACE] Tracing from `%s` to `%s` failed", node.Label, adjacent.Label)
			}
			*reports = append(*reports, report)
		} else if adjacent.Status == StatusDummy {
			hlog.CtxInfof(ctx, "[TRACE] `%s` is dummy node", adjacent.Label)
			trace(ctx, adjacent, reports)
		}
	}
}

func (g *Graph) Trace(ctx context.Context) {
	nodeDetails := []mongo.NodeDetail{}
	trace(ctx, g.Nodes["0"], &nodeDetails)
	totalNodes := 0
	successNodes := 0
	for _, node := range g.Nodes {
		if node.Status == StatusExecute {
			totalNodes += 1
		}
	}
	for _, node := range nodeDetails {
		if node.Result == mongo.RESULT_SUCCESS {
			successNodes += 1
		}
	}
	probeLog := mongo.ProbeLogModel{
		Id:           primitive.NewObjectID(),
		TotalNodes:   totalNodes,
		SuccessNodes: successNodes,
		NodesDetail:  nodeDetails,
		CreateTime:   time.Now(),
		UpdateTime:   time.Now(),
	}
	mongo.NewProbeLogModelDao().Save(ctx, probeLog)
}

// AddNode adds a new node to the graph
func (g *Graph) AddNode(id string) *Node {
	if _, exists := g.Nodes[id]; !exists {
		node := &Node{Id: id}
		g.Nodes[id] = node
	}
	return g.Nodes[id]
}

// AddEdge adds an edge between two nodes in the graph
func (g *Graph) AddEdge(from, to string) {
	node1 := g.AddNode(from)
	node2 := g.AddNode(to)
	node1.Adjacent = append(node1.Adjacent, node2)
}

// SetFunction sets the function for a node in the graph
func (g *Graph) SetFunction(id string, function Action) {
	if node, exists := g.Nodes[id]; exists {
		node.Function = function
	}
}

func (g *Graph) SetLabel(id, label string) {
	if node, exists := g.Nodes[id]; exists {
		node.Label = label
	}
}

func (g *Graph) SetStatus(id string, status int) {
	if node, exists := g.Nodes[id]; exists {
		node.Status = status
	}
}

// PrintGraph prints the graph
func (g *Graph) PrintGraph() {
	for _, node := range g.Nodes {
		fmt.Printf("Node %10v:", node.Label)
		for _, adjacent := range node.Adjacent {
			fmt.Printf(" %10v", adjacent.Label)
		}
		fmt.Println()
	}
}

type GraphConfig struct {
	Edges []struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"edges"`
	Functions map[string]struct {
		//Id            string `json:"id"`
		FunctionName string `json:"f_name"`
		Label        string `json:"label"`
		Status       int    `json:"status"`
	} `json:"functions"`
}

// LoadGraphFromConfig loads a graph from a configuration file
func LoadGraphFromConfig(filename string, functionMap map[string]func(ctx context.Context) bool) (*Graph, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	byteValue, _ := io.ReadAll(file)

	config := GraphConfig{}
	err = json.Unmarshal(byteValue, &config)
	if err != nil {
		return nil, err
	}

	graph := &Graph{Nodes: make(map[string]*Node)}

	for _, edge := range config.Edges {
		f1, exist := functionMap[config.Functions[edge.From].FunctionName]
		if !exist {
			continue
		}
		f2, exist := functionMap[config.Functions[edge.To].FunctionName]
		if !exist {
			continue
		}
		graph.AddEdge(edge.From, edge.To)

		graph.SetFunction(edge.From, f1)
		graph.SetLabel(edge.From, config.Functions[edge.From].Label)

		graph.SetFunction(edge.To, f2)
		graph.SetLabel(edge.To, config.Functions[edge.To].Label)
	}
	for id, function := range config.Functions {
		graph.SetStatus(id, function.Status)
	}

	return graph, nil
}

func RegisterFunctions() map[string]func(ctx context.Context) bool {
	functionMap := make(map[string]func(ctx context.Context) bool)
	obj := NewAgent()
	objValue := reflect.ValueOf(obj)
	objType := reflect.TypeOf(obj)
	for i := 0; i < objType.NumMethod(); i++ {
		method := objType.Method(i)
		functionMap[method.Name] = objValue.Method(i).Interface().(func(ctx context.Context) bool)
	}
	return functionMap
}
