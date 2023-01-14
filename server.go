package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"runtime"
	"sync"
	"time"
)

type Server struct {
	Ip               string
	Port             uint16
	OnlineUserMap    map[string]*User // 在线用户的列表,临界资源
	MapLock          sync.RWMutex     // 对临界资源加读写锁
	BroadCastMessage chan string      // 广播消息的管道
}

func NewServer(ip string, port uint16) *Server {
	Server := &Server{
		Ip:               ip,
		Port:             port,
		OnlineUserMap:    make(map[string]*User),
		BroadCastMessage: make(chan string),
	}
	return Server
}

// 监听广播消息的channel，一旦该channel有数据就广播发送
func (this *Server) ListeningBroadCast() {
	for {
		msg := <-this.BroadCastMessage

		// 有消息就将mes发送给所有的在线用户
		this.MapLock.RLock()
		for _, cli := range this.OnlineUserMap {
			cli.C <- msg
		}
		this.MapLock.RUnlock()
	}
}

// 服务器广播的方法
func (this *Server) BroadCast(user *User, mes string) {
	sendMes := "[" + user.Addr + "]" + user.Name + ":" + mes
	this.BroadCastMessage <- sendMes
}

func (this *Server) Handler(cli_conn net.Conn) {
	// 实例化一个User
	cur_User := NewUser(cli_conn, this)
	// 广播用户上线的消息
	cur_User.OnlineMes()
	isActive := make(chan bool)
	// 该协程监听各个客户端向服务端发送的消息
	go func() {
		for {
			buf := make([]byte, 4096)
			n, err := cur_User.conn.Read(buf)
			if n == 0 {
				cur_User.Offline()
				return
			}

			if err != nil && err != io.EOF {
				log.Fatal(err)
			}

			/*由于没有写客户端 暂时这样判断,防止客户端只发送\r\n*/
			if n < 3 {
				continue
			}

			msg := buf[:n-2]
			cur_User.DoMessage(string(msg))
			isActive <- true
		}
	}()

	// 超时强踢功能
	for {
		select {
		case <-isActive:
		case <-time.After(time.Minute * 3):
			cur_User.SendMesToCli("过久未响应，已与服务器断开连接！")
			cur_User.Offline()
			runtime.Goexit() // 退出当前协程
		}
	}

}

func (this *Server) Start() {
	// socket listen
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", this.Ip, this.Port))
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	// 启动监听广播信道的goroutine,一旦该channel有数据就广播发送
	go this.ListeningBroadCast()

	// accept
	for {
		cli, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go this.Handler(cli)
	}
}
