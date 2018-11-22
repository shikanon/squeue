package queue

import (
	"fmt"
	"net"
	"log"
	"encoding/binary"
	"sync"
)

type mockClient struct{
	sync.RWMutex
	conn net.Conn
}

func CreateMockClient(address string) (*mockClient, error){
	dstServer, err := net.Dial("tcp", address)
	if err != nil{
		return nil, err
	}
	return &mockClient{
		conn: dstServer, 
	}, nil
}

// 客户端握手协议
func (m *mockClient)Handshake()(ok bool){
	ok = false
	m.conn.Write([]byte("v01"))
	buff := make([]byte,2)
	n, err := m.conn.Read(buff)
	if err != nil || n != 2{
		log.Fatalf("握手协议错误 - %s", err)
		return
	}
	if string(buff)=="ok"{
		ok = true
		return
	}else{
		return
	}
}

func (m *mockClient)Write(info string) (error){
	bodyLen := len([]byte(info))
	buff := make([]byte, 4)
	binary.BigEndian.PutUint32(buff, uint32(bodyLen))

	head := "PUT " + string(buff) + "\n"
	m.conn.Write([]byte(head))
	// log.Println("写入",info)
	m.conn.Write([]byte(info))


	return nil
}

func (m *mockClient)Read()([]byte, error){
	body := "GET \n"
	m.conn.Write([]byte(body))
	return m.get()
}

func (m *mockClient)get()([]byte, error){
	var err error

	// 内容大小
	buff := make([]byte, 4)
	m.conn.Read(buff)
	size := binary.BigEndian.Uint32(buff)
	bodySize := size - 4

	// frameType
	m.conn.Read(buff)
	frameType := binary.BigEndian.Uint32(buff)

	// 内容
	body := make([]byte, bodySize)
	m.conn.Read(body)

	switch frameType{
	case errorFrameType:
		err = fmt.Errorf("错误：%s", string(body))
		return nil, err
	default:
		return body, nil
	}
}

func (m *mockClient)Close(){
	m.conn.Close()
}