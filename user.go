package main

import (
	"net"
	"strings"
)

type User struct {
	Name   string
	Addr   string
	C      chan string
	conn   net.Conn
	Server *Server
	isExit bool
}

// 创建一个用户的工厂函数
func NewUser(conn net.Conn, server *Server) *User {
	User := &User{
		Name:   conn.RemoteAddr().String(),
		Addr:   conn.RemoteAddr().String(),
		C:      make(chan string),
		conn:   conn,
		Server: server,
		isExit: false,
	}

	// 监听当前User Channel的数据，一旦有信息就直接发送给客户端，若没有信息则阻塞
	go func() {
		for message := range User.C {
			conn.Write([]byte(message + "\r\n"))
		}
	}()

	return User
}

func (this *User) OnlineMes() {
	this.Server.MapLock.Lock()
	this.Server.OnlineUserMap[this.Name] = this
	this.Server.MapLock.Unlock()
	this.Server.BroadCast(this, "上线了")
}

func (this *User) Offline() {
	this.Server.MapLock.Lock()
	defer this.Server.MapLock.Unlock()
	if this.isExit == true {
		return
	}

	delete(this.Server.OnlineUserMap, this.Name)

	close(this.C)
	this.conn.Close()
	this.isExit = true
	this.Server.BroadCast(this, "已下线")
}

// 将服务器的Reply发生给客户端
func (this *User) SendMesToCli(msg string) {
	this.conn.Write([]byte(msg + "\r\n"))
}

func (this *User) SendMesToPrivateChat(user *User, msg string) {
	user.conn.Write([]byte(msg + "\r\n"))
}

func (this *User) DoMessage(msg string) {
	// 私聊协议：to|[name]|[information]
	privateChat := func(msg string) (bool, []string) {
		info := strings.Split(msg, "|")
		if len(msg) > 4 && msg[:3] == "to|" && len(info) >= 3 && info[1] != "" {
			return true, info
		} else {
			return false, nil
		}
	}

	if msg == "/who" { // 查询在线列表逻辑
		this.Server.MapLock.Lock()
		var Msg_onlineUsers string
		for _, user := range this.Server.OnlineUserMap {
			Msg_onlineUsers += "[" + user.Addr + "]" + user.Name + ":" + "在线\r\n"
		}
		this.Server.MapLock.Unlock()
		this.SendMesToCli(Msg_onlineUsers[:len(Msg_onlineUsers)-1])
	} else if len(msg) > 7 && msg[:7] == "rename|" { // 修改用户名逻辑
		newName := strings.Split(msg, "|")[1]
		for name := range this.Server.OnlineUserMap {
			if name == newName {
				this.SendMesToCli("该用户名已存在.")
				return
			}
		}

		// 用户名可用，设置用户名
		this.Server.MapLock.Lock()
		delete(this.Server.OnlineUserMap, this.Name)
		this.Server.OnlineUserMap[newName] = this
		this.Server.MapLock.Unlock()
		this.Name = newName
		this.SendMesToCli("用户名修改成功！")

	} else if ok, targetUserInfo := privateChat(msg); ok {
		this.Server.MapLock.RLock()
		defer this.Server.MapLock.RUnlock()
		for name, user := range this.Server.OnlineUserMap {
			if name == targetUserInfo[1] {
				this.SendMesToPrivateChat(user, "[From"+this.Name+"]"+strings.Join(targetUserInfo[2:], "|"))
				return
			}
		}
		this.SendMesToCli("未找到当前用户")
	} else {
		this.Server.BroadCast(this, msg)
	}
}
