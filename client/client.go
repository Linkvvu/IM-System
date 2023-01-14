package main

import (
	"flag"
	"fmt"
	"log"
	"net"
)

type client struct {
	ServerPort uint16
	ServerIp   string
	Name       string
	conn       net.Conn
}

var (
	serverPort int
	serverIp   string
)

func init() {
	flag.IntVar(&serverPort, "port", 1993, "设置服务器Port")
	flag.StringVar(&serverIp, "ip", "127.0.0.1", "设置服务器Ip地址")
}

func NewClient(ip string, port uint16) *client {
	client := &client{
		ServerPort: port,
		ServerIp:   ip,
		conn:       nil,
		Name:       "",
	}
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		log.Print(err)
		return nil
	}

	client.conn = conn
	return client
}

func main() {
	flag.Parse() // 命令行参数解析

	client := NewClient(serverIp, uint16(serverPort))
	if client == nil {
		log.Fatal(">>>>>服务器连接失败...")
	}
	log.Println(">>>>>登陆成功...")

	scene := NewView()
	scene.run()
}
