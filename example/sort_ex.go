package main

import (
	"fmt"
	"sort"
)

// 下面是一个使用sort包对学生成绩排序的示例

// 学生成绩结构体
type StuScore struct {
	name string
	score int
}

type StuScores []StuScore

func (s StuScores) Len() int { return len(s) }
func (s StuScores) Less(i, j int) bool { return s[i].score < s[j].score } // 顺序
func (s StuScores) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func main()  {
	stus := StuScores{
		{"alan", 95},
		{"hikerell", 91},
		{"leao", 90},
		{"acmfly", 96}}

	fmt.Println("===Default===")
	//原始顺序
	for _, v := range stus {
		fmt.Println(v.name, ":",  v.score)
	}
	fmt.Println()
	// StuScores已经实现了sort.Interface接口
	sort.Sort(sort.Reverse(stus))

	fmt.Println("===Sorted===")
	//排好序后的结构
	for _, v := range stus {
		fmt.Println(v.name, ":", v.score)
	}

	//判断是否已经排好顺序，将会打印true
	fmt.Println("Is Sorted?", sort.IsSorted(stus))
}
