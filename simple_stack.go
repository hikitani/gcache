package gcache

import (
	"fmt"
	"sync"
)

var errStackIsEmpty = fmt.Errorf("stack is empty")

type stack struct {
	c []int

	sync.RWMutex
}

func newStack() *stack {
	s := new(stack)
	s.c = make([]int, 0, 10)

	return s
}

func (s *stack) Push(v int) {
	s.Lock()
	s.c = append(s.c, v)
	s.Unlock()
}

func (s *stack) Top() (int, error) {
	if s.IsEmpty() {
		return 0, errStackIsEmpty
	}

	s.RLock()
	v := s.c[len(s.c)-1]
	s.RUnlock()

	return v, nil
}

func (s *stack) Pop() (int, error) {
	top, err := s.Top()
	if err != nil {
		return top, err
	}

	lastIdx := s.Size() - 1
	s.Lock()
	s.c = s.c[:lastIdx]
	s.Unlock()

	return top, nil
}

func (s *stack) IsEmpty() bool {
	s.RLock()
	b := len(s.c) == 0
	s.RUnlock()
	return b
}

func (s *stack) Size() int {
	s.RLock()
	v := len(s.c)
	s.RUnlock()
	return v
}
