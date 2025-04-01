package example_pkg

import (
	"bufio"
	"errors"
	"example_pkg/gen"
	"fmt"
	"log"
	"net"
	"syscall"
	"time"
)

func client(name string, namePeer string, address string, message string) error {
	var conn net.Conn
	var err error
	for {
		conn, err = net.Dial("tcp", address)
		if err != nil {
			if errors.Is(err, syscall.ECONNREFUSED) {
				// 超时，重试
				time.Sleep(time.Duration(1000) * time.Millisecond)
			} else {
				fmt.Println("Error connecting:", err.Error())
				return err
			}
		} else {
			break
		}
	}

	defer func(conn net.Conn) {
		_ = conn.Close()
	}(conn)

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	nameMsg := &gen.Name{Name: name}
	err = writeMsg(name, "", writer, nameMsg)
	if err != nil {
		return err
	}

	// 创建Protobuf消息
	msg := &gen.MyMessage{Content: message}

	watchAppendMessage(fmt.Sprintf("%s %s", name, msg))
	err = writeMsg(name, namePeer, writer, msg)
	if err != nil {
		return err
	}

	// 读取响应
	err = readMsg(name, namePeer, reader, msg)
	if err != nil {
		return err
	}
	log.Println(name, "receives response from peer", namePeer, "message:", msg.Content)
	return nil
}
