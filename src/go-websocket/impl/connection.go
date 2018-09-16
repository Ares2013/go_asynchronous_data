package impl

import (
	"errors"
	"github.com/gorilla/websocket"
	"sync"
)

type Connection struct {
	wsConn *websocket.Conn
	inChan chan []byte
	outChan chan []byte
	closeChan chan byte

	mutex sync.Mutex
	isClosed bool
}

func InitConnection(wsConn *websocket.Conn)(conn *Connection,err error){
	conn = &Connection{
		wsConn:wsConn,
		inChan:make(chan []byte,1000),
		outChan:make(chan []byte,1000),
		closeChan:make(chan byte,1),
	}
	// 启动读协程
	go conn.readLoop()

	// 启动写协程
	go conn.writeLoop()
	return
}
// API
func (conn *Connection)ReadMessage()(data []byte,err error)  {
	select {
	case data = <-conn.inChan:
	case <-conn.closeChan:
		err = errors.New("Connection is closed")
	}
	return
}
func (conn *Connection)WriteMessage(data []byte)(err error)  {
	select {
	case conn.outChan<-data:
	case <-conn.closeChan:
		err = errors.New("Connection is closed")
	}

	return
}
func (conn *Connection)Close(){
	// 线程安全 可重复的close
	conn.wsConn.Close()
	// 要保证这个唯一性，只执行一次
	conn.mutex.Lock()
	if !conn.isClosed {
		close(conn.closeChan)
		conn.isClosed = true
	}
	conn.mutex.Unlock()
}

// 内部实现
func (conn *Connection)readLoop(){
	var(
		data []byte
		err error
	)
	for{
		if _,data,err = conn.wsConn.ReadMessage();err != nil {
			goto ERR
		}
		// 容量是1000，超过1000阻塞在这里，等待inChan有空闲
		select {
		case conn.inChan <- data:
		case <-conn.closeChan:
			// closeChan关闭的时候 会跳出程序
			goto ERR
		}
	}
ERR:
	conn.Close()
}
func (conn *Connection)writeLoop(){
	var(
		data []byte
		err error
	)
	for{
		select {
		case data=<-conn.outChan:
		case <-conn.closeChan:
			goto ERR
		}
		data = <-conn.outChan
		if err = conn.wsConn.WriteMessage(websocket.TextMessage,data);err != nil{
			goto ERR
		}
	}
ERR:
	conn.Close();
}