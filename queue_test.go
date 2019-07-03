package queue

import (
	"github.com/stretchr/testify/assert"
	"log"
	"testing"

)

func TestCreateDefaultQueue(t *testing.T){
	q := NewDefaultQueue(127)
	assert.Equal(t, len(q.data), 128)
	assert.Equal(t, q.mask+1, uint64(128))
}

func TestDefaultQueuePutAndGet(t *testing.T){
	q := NewDefaultQueue(127)
	q.Put("shikanon")
	value, err := q.Get()
	if err != nil{
		log.Fatal(err)
	}
	assert.Equal(t, value, "shikanon")
}

func TestDefaultQueueMax(t *testing.T){
	q := NewDefaultQueue(127)
	for i:=0; i<128; i++{
		q.Put("shikanon")
	}
	assert.Equal(t, q.IsFull(), true)

	for i:=0; i<128; i++{
		_, err := q.Get()
		if err != nil{
			log.Fatal(err)
		}
	}
	assert.Equal(t, q.IsEmpty(), true)
}