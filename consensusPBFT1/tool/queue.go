package tool

import (
	"container/list"
	"fmt"
	"sync"
)

const N int = 10

type Queue struct {
	figure  int
	digits1 [N]int
	digits2 [N]int
	sFlag   bool
	data    *list.List
}

var lock sync.Mutex

func NewQueue() *Queue {
	q := new(Queue)
	q.data = list.New()
	return q
}

func (q *Queue) Push(v interface{}) {
	defer lock.Unlock()
	lock.Lock()
	q.data.PushFront(v)
}

func (q *Queue) Dump() {
	for iter := q.data.Back(); iter != nil; iter = iter.Prev() {
		fmt.Println("item:", iter.Value)
	}
}

func (q *Queue) Pop() interface{} {
	defer lock.Unlock()
	lock.Lock()
	iter := q.data.Back()
	v := iter.Value
	q.data.Remove(iter)
	return v
}

func (q *Queue) Len() int {
	return q.data.Len()
}
