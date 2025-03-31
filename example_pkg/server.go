package example_pkg

import (
	"bufio"
	"example_pkg/gen"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

func server(name string, port uint16, ch *chan *sync.WaitGroup) error {
	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Println("Error listening:", err.Error())
		return err
	}
	wgServerClosed := &sync.WaitGroup{}
	wgServerClosed.Add(1)
	defer wgServerClosed.Done()

	go func(listen net.Listener) {
		if ch != nil {
			wg0 := <-*ch
			_ = listen.Close()
			wgServerClosed.Wait()
			wg0.Done()
		}
	}(listen)

	log.Printf("Server %s is listening on port %d...\n", name, port)
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Println("Error accepting:", err.Error())
			break
		}
		go handleClient(name, conn)
	}

	return nil
}

func handleClient(name string, conn net.Conn) {
	err := _handleClient(name, conn)
	if err != nil {
		log.Println(name, "error handling client:", err.Error())
	}
}

func _handleClient(name string, conn net.Conn) error {
	defer func(conn net.Conn) {
		_ = conn.Close()
	}(conn)

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for {
		msg := &gen.MyMessage{}
		err := readMsg(name, reader, msg)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		log.Println(name, "received message: ", msg.Content)

		watchAppendMessage(fmt.Sprintf("%s %s", name, msg))
		// 将消息内容回显给客户端
		response := &gen.MyMessage{Content: msg.Content}
		err = writeMsg(name, writer, response)
		if err != nil {
			return err
		}
	}
}
