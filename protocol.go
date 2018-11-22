package queue

import(
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	// "github.com/json-iterator/go"
)

const(
	errorFrameType 		uint32 = 0
	responseFrameType 	uint32 = 1
	messageFrameType	uint32 = 2
)

var maxBodyLen = 1 * 1024 * 1024 // 1Mb
var separatorBytes = []byte(" ")
// var json = jsoniter.ConfigCompatibleWithStandardLibrary

type Protocol interface{
	Execute() error
}

type proto struct {
	sync.RWMutex
	conn	net.Conn
	maxsize	int
	queue 	*DefaultQueue
}

/*
协议说明：
1. 客户端请求符合:
PUT \n
内容长度 \n
内容 \n

2. 方法：
3.1 PUT - 提交数据:
	PUT \n
	内容长度 \n
	内容 \n
example：
	PUT \n
	10 \n
	content \n

3.2 GET - 获取数据
3.3 FIN - 结束对话
*/
func (p *proto)Execute() error{
	var err error
	var line []byte
	reader := bufio.NewReaderSize(p.conn, p.maxsize)
	for {
		// 通过换行符分割
		line, err = reader.ReadSlice('\n')
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			break
		}
		// 去掉换行符
		line = line[:len(line)-1]
		if len(line) < 2{
			err = fmt.Errorf("协议错误 - 不符合协议要求, 请求内容为%d, 长度小于3!", len(line))
			p.SentErrorMessage(err)
			return err
		}

		// 决定具体使用哪个方法
		params := bytes.Split(line, separatorBytes)
		switch {
		case bytes.Equal(params[0], []byte("FIN")):
			return nil
		case bytes.Equal(params[0], []byte("PUT")):
			p.PUT(params)
		case bytes.Equal(params[0], []byte("GET")):
			p.GET()
		default:
			return nil
		}
		if p.queue.IsFull(){
			log.Println("队列已满...")
			p.queue.Dispose()
		}
		if p.queue.IsEmpty(){
			log.Println("队列是空的...")
		}
	}
	return nil
}

func (p *proto)PUT(params [][]byte) (err error){
	if len(params) < 1 {
		err = fmt.Errorf("PUT协议错误 - 参数个数为 %d, 参数长度不对!", len(params))
		p.SentErrorMessage(err)
		return err
	}
	// 内容长度
	bodyLen := int32(binary.BigEndian.Uint32(params[1]))
	if bodyLen < 0{
		return fmt.Errorf("PUT方法为无效的内容长度，%d", bodyLen)
	}
	// 内容
	body := make([]byte, bodyLen)
	_, err = io.ReadFull(p.conn, body)
	if err != nil{
		return err
	}

	p.queue.Put(body)
	return nil
}

func (p *proto)GET() (int, error){
	p.RLock()
	result, err := p.queue.Get()
	// log.Println("读出来的数据：", result)
	if err != nil{
		log.Fatal(err)
		return 0, nil
	}
	buff := &bytes.Buffer{}
	binary.Write(buff, binary.BigEndian, result)
	// log.Println("序列化",string(buff.Bytes()),buff.Bytes())

	n, err := p.SentResponse(buff.Bytes())
	p.RUnlock()
	return n,err
}

// 对消息进行回复,返回response
func (p *proto)SentResponse(data []byte) (int, error){
	return p.sent(data, responseFrameType)
}

// 发送消息,通知客户端
func (p *proto)SentRequest(data []byte) (int, error){
	return p.sent(data, messageFrameType)
}

// 发送错误信息
func (p *proto)SentErrorMessage(errorInfo error) (int, error){
	return p.sent([]byte(errorInfo.Error()), errorFrameType)
}

// 发送消息格式：
// 1.数据包大小
// 2.方法
// 3.数据
func (p *proto)sent(data []byte, frameType uint32)(int, error){
	// int32 , 4字节
	buff := make([]byte, 4)
	size := uint32(len(data)) + 4 //总大小

	// 将数据转换为二进制协议
	binary.BigEndian.PutUint32(buff, size)
	n, err := p.conn.Write(buff)
	if err != nil {
		return n, err
	}

	binary.BigEndian.PutUint32(buff, frameType)
	n, err = p.conn.Write(buff)
	if err != nil {
		return n + 4, err
	}

	n, err = p.conn.Write(data)
	return n + 8, err
}
