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
	s.RLock()
	if len(s.c) == 0 {
		s.RUnlock()
		return 0, errStackIsEmpty
	}

	v := s.c[len(s.c)-1]
	s.RUnlock()
	return v, nil
}

func (s *stack) Pop() (int, error) {
	s.Lock()
	if len(s.c) == 0 {
		s.Unlock()
		return 0, errStackIsEmpty
	}

	v := s.c[len(s.c)-1]
	s.c = s.c[:len(s.c)-1]
	s.Unlock()
	return v, nil
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
