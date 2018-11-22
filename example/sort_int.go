package main

import (
	"fmt"
	"sort"
)

type IntHeap []int

func (i IntHeap) Len() int { return len(i) }
func (i IntHeap) Less(j, k int) bool { return i[j] > i[k] } // 顺序
func (i IntHeap) Swap(j, k int) { i[j], i[k] = i[k], i[j] }

func main() {
	var intheap = IntHeap{3, 6, 2, 9}
	for _, v := range intheap {
		fmt.Println(v)
	}

	fmt.Println()

	sort.Sort(intheap)
	for _, v := range intheap {
		fmt.Println(v)
	}
}