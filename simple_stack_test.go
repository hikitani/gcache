package gcache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleStack(t *testing.T) {
	var (
		v   int
		err error
	)

	s := newStack()

	assert.Equal(t, 0, len(s.c))
	assert.Equal(t, true, s.IsEmpty())

	v, err = s.Top()
	assert.Equal(t, 0, v)
	assert.Equal(t, errStackIsEmpty, err)

	v, err = s.Pop()
	assert.Equal(t, 0, v)
	assert.Equal(t, errStackIsEmpty, err)

	s.Push(1)
	v, err = s.Top()
	assert.Equal(t, 1, v)
	assert.Nil(t, err)

	v, err = s.Pop()
	assert.Equal(t, 1, v)
	assert.Nil(t, err)
	assert.Equal(t, true, s.IsEmpty())

	s.Push(1)
	s.Push(2)
	s.Push(3)

	assert.Equal(t, 3, s.Size())
	v, err = s.Pop()
	assert.Equal(t, 3, v)
	assert.Nil(t, err)
	v, err = s.Pop()
	assert.Equal(t, 2, v)
	assert.Nil(t, err)
	v, err = s.Pop()
	assert.Equal(t, 1, v)
	assert.Nil(t, err)
}
