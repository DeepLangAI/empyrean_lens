package link_trace

import (
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestTimeParse(t *testing.T) {
	t1, e := time.Parse("2006-01-02", "1111")
	if e != nil {
		t.Errorf("time parse err:%v", e)
	}
	t.Log(t1.IsZero())
	t1, e = time.Parse("2006-01-02", "2024-11-23")
	if e != nil {
		t.Errorf("time parse err:%v", e)
	}
	t.Log(t1.IsZero())
}

func TestSyncMap(t *testing.T) {
	sm := &sync.Map{}
	sm.Store(1, []string{"1", "2", "3"})
	if v, ok := sm.Load(1); ok {
		s := strings.Join(v.([]string), "/")
		t.Log(s)
	}
}

// 测试函数
func TestBubbleSort(t *testing.T) {
	// 测试切片排序
	arr := []int{5, 4, 3, 2, 1}
	less := func(i, j int) bool {
		return arr[i] < arr[j]
	}
	bubbleSort(arr, less)
	expected := []int{1, 2, 3, 4, 5}
	if !reflect.DeepEqual(arr, expected) {
		t.Errorf("排序结果不正确，期望 %v，实际 %v", expected, arr)
	}

	// 测试空切片
	arr = []int{}
	bubbleSort(arr, less)
	expected = []int{}
	if !reflect.DeepEqual(arr, expected) {
		t.Errorf("空切片排序结果不正确，期望 %v，实际 %v", expected, arr)
	}

	// 测试单个元素切片
	arr = []int{1}
	bubbleSort(arr, less)
	expected = []int{1}
	if !reflect.DeepEqual(arr, expected) {
		t.Errorf("单个元素切片排序结果不正确，期望 %v，实际 %v", expected, arr)
	}
}

func TestQuickSort(t *testing.T) {
	// 测试用例 1：空切片
	arr1 := []int{}
	quickSort(arr1, func(i, j int) bool { return arr1[i] < arr1[j] })
	if len(arr1) != 0 {
		t.Errorf("Expected empty slice, got %v", arr1)
	}

	// 测试用例 2：单元素切片
	arr2 := []int{1}
	quickSort(arr2, func(i, j int) bool { return arr2[i] < arr2[j] })
	if len(arr2) != 1 || arr2[0] != 1 {
		t.Errorf("Expected [1], got %v", arr2)
	}

	// 测试用例 3：已排序切片
	arr3 := []int{1, 2, 3, 4, 5}
	quickSort(arr3, func(i, j int) bool { return arr3[i] < arr3[j] })
	if !reflect.DeepEqual(arr3, []int{1, 2, 3, 4, 5}) {
		t.Errorf("Expected [1, 2, 3, 4, 5], got %v", arr3)
	}

	// 测试用例 4：逆序切片
	arr4 := []int{5, 4, 3, 2, 1}
	quickSort(arr4, func(i, j int) bool { return arr4[i] < arr4[j] })
	if !reflect.DeepEqual(arr4, []int{1, 2, 3, 4, 5}) {
		t.Errorf("Expected [1, 2, 3, 4, 5], got %v", arr4)
	}

	// 测试用例 5：包含重复元素的切片
	arr5 := []int{3, 1, 4, 1, 5, 9, 2, 6, 5, 3, 5}
	quickSort(arr5, func(i, j int) bool { return arr5[i] < arr5[j] })
	expected := []int{1, 1, 2, 3, 3, 4, 5, 5, 5, 6, 9}
	if !reflect.DeepEqual(arr5, expected) {
		t.Errorf("Expected %v, got %v", expected, arr5)
	}
}
