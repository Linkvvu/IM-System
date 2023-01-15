package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"syscall"
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

// TCP连接关闭error信息判断
func (this client) judgeConnError(err error) bool {
	netErr, ok := err.(net.Error)
	if !ok {
		return false
	}

	if netErr.Timeout() {
		log.Println("timeout..")
		return true
	}

	opErr, ok := netErr.(*net.OpError)
	if !ok {
		return false
	}

	switch typ := opErr.Err.(type) {
	case *net.DNSError:
		log.Printf("net.DNSError:%+v", typ)
		return true
	case *os.SyscallError:
		log.Printf("os.SyscallError:%+v", typ)

		if errno, ok := typ.Err.(syscall.Errno); ok {
			switch errno {
			case syscall.ECONNREFUSED:
				log.Println("connect refused")
				return true
			case syscall.ETIMEDOUT:
				log.Println("timeout")
				return true
			case syscall.WSAECONNABORTED:
				log.Println("对端强制关闭连接,已与服务器断开连接")
				return false
			case syscall.WSAECONNRESET:
				//...
				return false
			}
		}
	}
	return false
}

func (this client) SendMsgToServer(msg string) error {
	_, err := this.conn.Write([]byte(msg + "\r\n"))
	return err
}

func (this *client) CloseConnection() {
	this.conn.Close()
}

func (this *client) Deal_With_Responses() {
	// io.Copy(os.Stdout, this.conn)	// 此方法不能感知tcp连接的状态
	for {
		responses := make([]byte, 4096)
		n, err := this.conn.Read(responses)
		if err == io.EOF { // 感知到对端已经关闭tcp连接
			this.CloseConnection() // 关闭本端tcp连接
			os.Exit(1)             // 退出客户端程序，Status Code 为 1
		}

		if n == 0 && !this.judgeConnError(err) { // 本端已经关闭tcp连接 或 对端强制关闭连接
			this.CloseConnection() // 关闭本端tcp连接
			os.Exit(1)             // 退出客户端程序，Status Code 为 1
			// ......
			// return
		}

		if err != nil && err != io.EOF {
			log.Println(err)
		} else {
			fmt.Print(string(responses[:n]))
		}

	}
}

func (this *client) ModifyUserName() {
	fmt.Printf(">>>>>请输入用户名： ")
	fmt.Scanln(&this.Name)
	msg := "rename|" + this.Name
	err := this.SendMsgToServer(msg)
	if err != nil {
		log.Println(err)
	}
}

func (this client) readLine(Msg *string) error { // 读取整行信息函数
	reader := bufio.NewReader(os.Stdin)
	msg, err := reader.ReadString('\n')
	*Msg = msg
	return err
}

func (this *client) selectUsers() error { // 查询在线用户
	sendMsg := "/who"
	err := this.SendMsgToServer(sendMsg)
	return err
}

func (this *client) privateChat() {
	var content string
	var targetUser string
	var finalMsg string
	err := this.selectUsers()
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println(">>>请输入聊天对象[用户名]（/exit退出）")
	err = this.readLine(&targetUser)
	if err != nil {
		log.Println(err)
		return
	}
	targetUser = targetUser[:len(targetUser)-2]

	for targetUser != "/exit" {
		fmt.Println(">>>请输入聊天内容")
		err = this.readLine(&content)
		if err != nil {
			log.Println(err)
			return
		}

		finalMsg = "to|" + targetUser + "|" + content // 组合私聊协议的格式
		_, err = this.conn.Write([]byte(finalMsg))
		if err != nil {
			log.Println(err)
			return
		}

		fmt.Println(">>>请输入聊天对象[用户名]（/exit退出）")
		err = this.readLine(&targetUser)
		if err != nil {
			log.Println(err)
			return
		}
		targetUser = targetUser[:len(targetUser)-2]
	}
}

func (this *client) publicChar() {
	// 提示用户输入
	var chatMsg string

	fmt.Println(">>>请输入聊天内容(/exit退出)")
	err := this.readLine(&chatMsg)
	if err != nil {
		log.Println(err)
		return
	}

	for chatMsg != "/exit\r\n" {
		// 消息不为空则发生
		if len(chatMsg) != 0 {
			_, err := this.conn.Write([]byte(chatMsg))
			if err != nil {
				log.Println(err)
				break
			}

			fmt.Println(">>>请输入聊天内容(/exit退出)")
			err = this.readLine(&chatMsg)
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}

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

	go client.Deal_With_Responses() // 开启一个goroutine去读取客户端的回应

	scene := NewView(client)
	scene.run()
}
