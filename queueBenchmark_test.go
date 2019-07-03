package queue

import (
	"testing"
	"sync"
)

/*
SPSC(single producers single consumer)：一个Producer ：一个Consumer
SPMC(single producers multiple consumer)：一个Producer ：多个Consumer
MPSC(multiple producers single consumer)：多个Producer ：一个Consumer
MPMC(multiple producers multiple consumer)：多个Producer ：多个Consumer
*/
func BenchmarkQueueMPSC(b *testing.B){
	var testByte []byte
	for i:= 0; i<100; i++{
		testByte = append(testByte,byte(i))
	}

	q := NewDefaultQueue(uint64(b.N))
	b.ResetTimer() //基准测试在运行之前需要昂贵的设置，计时器需要重置
	go func() {
		for i := 0; i < b.N; i++ {
			q.Put(testByte)
		}
	}()

	for i := 0; i < b.N; i++ { 
		q.Get()
	}
}

func BenchmarkQueueMPMC(b *testing.B){
	var testByte []byte
	for i:= 0; i<100; i++{
		testByte = append(testByte,byte(i))
	}

	runNums := 100
	q := NewDefaultQueue(uint64(b.N*runNums))
	var wg sync.WaitGroup
	wg.Add(runNums)
	b.ResetTimer() //计时器需要重置

	for i := 0; i < runNums; i++ {
		go func() {
			for i := 0; i < b.N; i++ {
				q.Put(testByte)
			}
		}()
	}

	for i := 0; i < runNums; i++ {
		go func() {
			for i := 0; i < b.N; i++ {
				q.Get()
			}
			wg.Done()
		}()
	}

	wg.Wait()
}
