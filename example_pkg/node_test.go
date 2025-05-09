package example_pkg

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
	"sync"
	"testing"
)

func watchServer(port uint16) WatchContext {
	server := NewWatchServer(port)
	context := WatchContext{
		channel:  make(chan string),
		messages: make([]string, 0),
	}
	server.Register("/trace_message", &context.channel, &context,
		func(writer http.ResponseWriter, request *http.Request) {
			uri := request.RequestURI
			s := server.context[uri]
			if s != nil {
				ctx := reflect.ValueOf(s).Interface().(*WatchContext)
				if ctx != nil {
					for _, m := range ctx.messages {
						_, err := writer.Write([]byte(fmt.Sprintf("%s\n", m)))
						if err != nil {
							log.Fatal(err)
						}
					}
				}
			}
		},
		func(channel interface{}, object interface{}) {
			if channel != nil && object != nil {
				channel := reflect.ValueOf(channel).Interface().(*chan string)
				ctx := reflect.ValueOf(object).Interface().(*WatchContext)
				if ctx != nil && channel != nil {
					for {
						s := <-*channel
						ctx.messages = append(ctx.messages, s)
					}
				}
			}
		},
	)
	go server.Serve()
	return context
}

func runNodes(
	n int, // N个节点
) {
	context := watchServer(9099)
	setWatchCtx(&context)

	name2addr := make(
		map[string]Address)
	name2chan := make(
		map[string]chan *sync.WaitGroup)
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("n_%d", i+1)
		ip := "127.0.0.1"
		port := uint16(8080) + uint16(i) + 1
		name2addr[name] = Address{
			IP:   ip,
			Port: port,
		}
		name2chan[name] = make(chan *sync.WaitGroup, 1)
	}

	chDone := make(chan bool, 1)
	for name := range name2addr {
		//只有n1接收输入
		isClient := name == "n_1"
		go func(nodeName string,
			name2addr map[string]Address,
			name2chan map[string]chan *sync.WaitGroup,
			isClient bool) {
			ch := name2chan[name]
			node(name, name2addr, isClient, &chDone, &ch)
		}(name, name2addr, name2chan, isClient)
	}
	_ = <-chDone
	for _, ch := range name2chan {
		wg := &sync.WaitGroup{}
		wg.Add(1)
		ch <- wg
		wg.Wait()
	}
}

func FuzzMessage(f *testing.F) {
	f.Fuzz(func(
		t *testing.T,
		n int64,
	) {
		t.Log("fuzz testing seed ", n)
		CreateFuzz(n)
		runNodes(10)
	})
}
