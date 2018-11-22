package queue

import (
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)



func TestTCPHandler(t *testing.T){
	log.Println("开始启动客户端进行测试...")
	c,err := CreateMockClient("127.0.0.1:8888")
	if err !=nil {
		log.Fatalf("创建mock连接错误 - %s", err)
	}
	ok := c.Handshake()
	if !ok{
		log.Fatalln("握手协议错误，握手失败!")
	}

	for i := 0; i<100; i++{
		c.Write("shikanon")
		result,err := c.Read()
		if err !=nil{
			log.Fatalf("读取错误: %s", err)
		}
	
		assert.Equal(t, string(result), "shikanon")
	}

	c.Close()
}
