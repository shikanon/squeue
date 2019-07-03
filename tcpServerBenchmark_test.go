package queue

import (
	// "github.com/stretchr/testify/assert"
	// "fmt"
	"log"
	"net"
	"sync"
	"testing"
	"time"
	// "os"
	// "runtime/trace"
)

func init(){
	listener,err := net.Listen("tcp", ":8888")
	if err != nil {
		log.Fatalf("测试错误：tcp监听错误 - %s", err)
	}
	handler := &tcpServer{queue: NewDefaultQueue(2024)}
	go ListenServer(listener, handler)
	time.Sleep(time.Second*5)
}

/*
SPSC(single producers single consumer)：一个Producer ：一个Consumer
SPMC(single producers multiple consumer)：一个Producer ：多个Consumer
MPSC(multiple producers single consumer)：多个Producer ：一个Consumer
MPMC(multiple producers multiple consumer)：多个Producer ：多个Consumer
*/

func BenchmarkTCPHandlerMPMC(b *testing.B){

	runNums := 100
	log.Println(b.N, b.N/1000)

	log.Println("性能测试客户端启动...")
	c,err := CreateMockClient("127.0.0.1:8888")
	if err !=nil {
		log.Fatalf("创建mock连接错误 - %s", err)
	}
	log.Println("开始握手...")
	ok := c.Handshake()
	if !ok{
		log.Fatalln("握手协议错误，握手失败!")
	}

	var wg sync.WaitGroup
	wg.Add(runNums)
	b.ResetTimer() //计时器需要重置

	for i := 0; i < runNums; i++ {
		go func() {
			for i := 0; i < b.N/1000; i++ {
				c.Write("shikanon")
			}
		}()
	}

	for i := 0; i < runNums; i++ {
		go func() {
			for i := 0; i < b.N/1000; i++ {
				result,err := c.Read()
				if err != nil{
					log.Fatal(err)
				}
				if string(result) != "shikanon"{
					log.Fatalln("值错误！")
				} 
				// log.Println(n, i)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	c.Write("FIN \n")
	c.Close()
}
