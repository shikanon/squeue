package queue

import (
	"net"
	"log"
	"io"
)

type TCPHandler interface {
	Handle(net.Conn)
}

func ListenServer(listener net.Listener, handler TCPHandler) {
	log.Printf("TCP: listening on %s", listener.Addr())

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			log.Panicf("temporary Accept() failure - %s", err)
			break
		}
		go handler.Handle(clientConn)
	}

	log.Printf( "TCP: closing %s", listener.Addr())
}

type tcpServer struct {
	queue 	*DefaultQueue
}

/*
协议：
1、前3字节为魔法字符：表示版本号
2、确认后向客户端应答"ok"
3、开始通信
PUT - 提交数据
GET - 获取数据
FIN - 结束对话
*/
func (s *tcpServer)Handle(client net.Conn){
	defer client.Close()
	var err error
	buf := make([]byte, 3)
	_, err = io.ReadFull(client, buf)
	if err != nil{
		log.Fatalf("失败加载 - %s", err)
		return
	}
	version := string(buf)
	var p Protocol
	switch version{
	case "v01":
		p = &proto{
			conn: client, 
			maxsize: 16*1024,
			queue: s.queue,
			// data: s.data,
		}
	default:
		log.Fatalf("协议版本号不符合，连接关闭 - 错误版本号：%s", version)
		return
	}
	// 返回 "ok", 表示连接上，后面可以
	client.Write([]byte{111, 107})

	// 处理数据
	err = p.Execute()
	if err != nil{
		log.Fatalf("协议传输出错 - %s", err)
		return
	}
}
