package example_pkg

import (
	"container/heap"
	"google.golang.org/protobuf/proto"
	"log"
	"math/rand"
	"sync"
)

type FuzzContext struct {
	mu         *sync.Mutex
	name2queue map[string]*NodeQueue
	rand       *rand.Rand
}

type NodeQueue struct {
	mu    *sync.Mutex
	cond  *sync.Cond
	queue PriorityQueue
}

// An Item is something we manage in a priority queue.
type Item struct {
	value    []byte // The value of the item; arbitrary.
	priority uint   // The priority of the item in the queue.
	// The index is needed by update and is maintained by the heap.Interface methods.
	index int // The index of the item in the heap.
}

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*Item

func (pq *PriorityQueue) Len() int { return len(*pq) }

func (pq *PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the lowest, not highest, priority so we use less than here.
	return (*pq)[i].priority < (*pq)[j].priority
}

func (pq *PriorityQueue) Swap(i, j int) {
	(*pq)[i], (*pq)[j] = (*pq)[j], (*pq)[i]
	(*pq)[i].index = i
	(*pq)[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// update modifies the priority and value of an Item in the queue.
func (pq *PriorityQueue) update(item *Item, value []byte, priority uint) {
	item.value = value
	item.priority = priority
	heap.Fix(pq, item.index)
}

var _FuzzContext *FuzzContext = nil

func CreateFuzz(seed int64) {
	source := rand.NewSource(seed)
	mu := new(sync.Mutex)
	_FuzzContext = &FuzzContext{
		mu:         mu,
		rand:       rand.New(source),
		name2queue: make(map[string]*NodeQueue),
	}
}

const PercentLost = 0
const PercentRepeat = 0
const PercentDelay = 0

type FuzzInfo struct {
	priority uint
	delay    bool
	lost     bool
	repeat   bool
}

func FuzzGen(n uint, priority uint) FuzzInfo {
	lost := (n % 100) < PercentLost
	repeat := (n % 100) < PercentRepeat+PercentRepeat
	delay := (n % 100) < PercentRepeat+PercentRepeat+PercentDelay
	info := FuzzInfo{
		priority: priority,
		lost:     lost,
		repeat:   repeat,
		delay:    delay,
	}
	return info
}

func FuzzMsg(name string, msg proto.Message) error {
	if _FuzzContext != nil {
		bytes, e1 := proto.Marshal(msg)
		if e1 != nil {
			return e1
		}
		var queue *NodeQueue = nil
		_FuzzContext.mu.Lock()
		if q, ok := _FuzzContext.name2queue[name]; ok {
			queue = q
		} else {
			mu := new(sync.Mutex)
			cond := sync.NewCond(mu)
			queue = &NodeQueue{
				mu:    mu,
				cond:  cond,
				queue: make(PriorityQueue, 0),
			}

			_FuzzContext.name2queue[name] = queue
		}
		_FuzzContext.mu.Unlock()

		queue.mu.Lock()
		info := FuzzGen(uint(_FuzzContext.rand.Uint32()), uint(_FuzzContext.rand.Uint32()))
		if info.repeat {
			log.Println("repeat message", msg)
			queue.queue.Push(&Item{value: bytes, priority: 0})
			queue.queue.Push(&Item{value: bytes, priority: info.priority})
		} else if info.delay {
			log.Println("delay message", msg)
			queue.queue.Push(&Item{value: bytes, priority: info.priority})
		} else if !info.lost {
			queue.queue.Push(&Item{value: bytes, priority: 0})
		} else {
			log.Println("lost message", msg)
		}
		queue.mu.Unlock()
		queue.cond.Broadcast()
		empty := true
		for empty {
			queue.mu.Lock()
			empty = queue.queue.Len() == 0
			if empty {
				queue.cond.Wait()
			} else {
				item := queue.queue.Pop().(*Item)
				_bytes := item.value
				e2 := proto.Unmarshal(_bytes, msg)
				if e2 != nil {
					return e2
				}
			}
			queue.mu.Unlock()
		}
		return nil
	} else {
		return nil
	}
}
