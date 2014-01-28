package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"runtime"
	"strings"
	"time"
)

/****************************************************
*@brief 定义消息，所有收发消息都采用同样格式，并采用
json加密，
*****************************************************
*@param Cmd:消息指令，包括connect、login、list、group、chat、beat
*@param Data:消息内容
*@param Sender：发送者，在用户名域
*@param Receiver：接受者，在用户名域
*****************************************************/

type Message struct {
	Cmd      string
	Data     string
	Sender   string
	Receiver string
}

/****************************************************
*@brief 定义客户端用户
*****************************************************
*@param reader:消息接收接口
*@param remoteAddr:远程服务器地址
*@param name:用户名   
*@param chatPort：用户监听端口     
*****************************************************/
type User struct {
	reader       *net.UDPConn
	remoteClient *User
	name         string
	chatPort     string
}

/****************************************************
*@function Login(loginPort string) User
*****************************************************
*@brief 本地创建user，同时向服务器注册
*****************************************************
*@access Public
*****************************************************
*@param 无
*****************************************************
*@return user
*****************************************************/
func Login(loginPort string) User { //随机本地监听开启端口
	//获取本地ip地址
	addrs, _ := net.InterfaceAddrs()
	fmt.Println(addrs)
	localHost := addrs[0]
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	port := rand.Int() % 65536
	user := new(User)
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%v", localHost.String(), port))
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("localAddr:%v\n", udpAddr.String())
	listener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println(err)
	}
	user.reader = listener
	//向服务器注册端口发起注册
	loginConn, err := net.Dial("tcp", loginPort)
	if err != nil {
		fmt.Println(err)
	}
	defer loginConn.Close()
	mes := Message{
		Cmd:      "connect",
		Sender:   loginConn.LocalAddr().String(),
		Data:     udpAddr.String(),
		Receiver: "server",
	}
	//json加密发送消息
	coder := json.NewEncoder(loginConn)
	err = coder.Encode(mes)
	if err != nil {
		fmt.Println(err)
	}
	//获取服务器返回的chat、beat端口
	decoder := json.NewDecoder(loginConn)
	err = decoder.Decode(&mes)
	if err != nil {
		fmt.Println(err)
	}
	user.chatPort = mes.Data
	//fmt.Printf("CMD:%v,DATA:%v,Sender:%v,Receiver:%v\n", mes.Cmd, mes.Data, mes.Sender, mes.Receiver)
	user.remoteClient = &User{
		reader:       nil,
		remoteClient: nil,
		name:         "server",
		chatPort:     "",
	}
	//接收welcome介绍
	err = decoder.Decode(&mes)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(mes.Data)
	fmt.Println("1.list: used to list all users")
	fmt.Println("2. group: group XXX used to create a conversation between XXX")
	fmt.Println("3.quit:used to quit a conversation\n")
	//输入用户名,服务器端检查是否被使用
	flag := true
	buffer := make([]byte, 1024)
	for flag {
		count, err := os.Stdin.Read(buffer)
		if err != nil {
			fmt.Println(err)
		}
		user.name = strings.TrimSpace(strings.TrimSpace(string(buffer[:count])))
		if user.name == "list" || user.name == "group" || user.name == "quit" {
			fmt.Println("you can not use the keyword as your name")
		} else {
			mes = Message{
				Cmd:      "login",
				Sender:   loginConn.LocalAddr().String(),
				Data:     user.name,
				Receiver: "server",
			}
			//fmt.Printf("CMD:%v,DATA:%v,Sender:%v,Receiver:%v\n", mes.Cmd, mes.Data, mes.Sender, mes.Receiver)
			encoder := json.NewEncoder(loginConn)
			err = encoder.Encode(mes)
			if err != nil {
				fmt.Println(err)
			}
			err = decoder.Decode(&mes)
			//fmt.Printf("CMD:%v,DATA:%v,Sender:%v,Receiver:%v\n", mes.Cmd, mes.Data, mes.Sender, mes.Receiver)
			if err != nil {
				fmt.Println(err)
			}
			if mes.Data == "success" {
				fmt.Println("success to login")
				break
			} else {
				fmt.Println("the name has already been token,please try another name")
				mes.Sender, mes.Receiver = mes.Receiver, mes.Sender
			}
		}
	}
	//fmt.Println(*user)
	return *user
}

/****************************************************
*@function func (u *User) Read(groupCh chan string)
*****************************************************
*@brief 监听本地端口
*****************************************************
*@access Public
*****************************************************
*@param groupCh chan string 读写进程间group状态交互channel
*****************************************************
*@return 无
*****************************************************/
func (u *User) Read(groupCh chan string, logger *log.Logger) {
	decoder := json.NewDecoder(u.reader)
	var mess Message
	for {
		err := decoder.Decode(&mess)
		if err != nil {
			fmt.Println(err)
		}
		logger.Printf("read:%v\n", mess)
		now := fmt.Sprintf("%d:%d:%d", time.Now().Hour(), time.Now().Minute(), time.Now().Second())
		fmt.Printf("%s:", now)
		switch mess.Cmd {
		//list指令，显示所有在线用户名
		case "list":
			{
				fmt.Println("these online users are:")
				userlist := strings.Split(mess.Data, "/")
				for _, userName := range userlist {
					fmt.Println(userName)
				}
			}
		//group指令，收到后，保存会话用户
		case "group":
			{
				//fmt.Println(mess.Data)
				//mess.Data包含3部分数据：1.发起者(1)\接受者(0)标志；2.名字；3.udp地址
				groupList := strings.Split(mess.Data, "/")
				//fmt.Println(groupList)
				if groupList[0] == "0" {
					fmt.Printf("%s want talk to you\n", groupList[1])
					groupInfo := strings.Join(groupList[1:], "/")
					groupCh <- groupInfo
				} else if groupList[0] == "1" {
					fmt.Printf("now you can talk to %s\n", groupList[1])
					groupInfo := strings.Join(groupList[1:], "/")
					groupCh <- groupInfo
				} else {
					fmt.Println(mess.Data)
				}

			}
			//quit指令，收到后清除会话用户
		case "quit":
			{
				groupCh <- "quit"
			}
			//chat指令，收到后，显示会话内容
		case "chat":
			{
				fmt.Printf("<%s>:%s\n", mess.Sender, mess.Data)
			}
		}
	}
}

/****************************************************
*@function HeartBeat(beatAddr string, userName string)
*****************************************************
*@brief 本地发送心跳接口
*****************************************************
*@access Public
*****************************************************
*@param beatAddr string 服务器监听地址
*@param userName string 用户名
*****************************************************
*@return 无
*****************************************************/
func HeartBeat(beatAddr string, userName string) {
	udpAddr, _ := net.ResolveUDPAddr("udp", beatAddr)
	timer := time.NewTimer(1 * time.Second)
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		fmt.Println(err)
	}
	encoder := json.NewEncoder(conn)
	for {
		select {
		case <-timer.C:
			{
				mess := Message{
					Cmd:      "beat",
					Sender:   userName,
					Data:     "",
					Receiver: "server",
				}
				//fmt.Println(mess)
				err := encoder.Encode(mess)
				if err != nil {
					fmt.Println(err)
				}
			}
		}
		timer.Reset(1 * time.Second)
	}
}

/****************************************************
*@function Write(serverAddr string)
*****************************************************
*@brief 本地输入接口，绑定本地唯一发送端口
加开一个进程，用于发送心跳，维护在线状态
*****************************************************
*@access Public
*****************************************************
*@param 无
*****************************************************
*@return 无
*****************************************************/
func (u *User) Write(groupCh chan string, logger *log.Logger) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	//开启进程，定时发送心跳，维护在线
	go func(beatPort string, userName string) {
		HeartBeat(beatPort, userName)
	}(u.chatPort, u.name)
	//接收用户输入并发送
	inputCh := make(chan string)
	go func(ch chan string) {
		buffer := make([]byte, 1024)
		for {
			fmt.Printf("<%s>:", u.name)
			count, err := os.Stdin.Read(buffer)
			if err != nil {
				fmt.Println(err)
			}
			str := strings.TrimSpace(strings.TrimSpace(string(buffer[:count])))
			ch <- str
		}
	}(inputCh)
	var mess Message
	chatUdpAddr, _ := net.ResolveUDPAddr("udp", u.chatPort)
	chatConn, err := net.DialUDP("udp", nil, chatUdpAddr)
	for {
		//并发逻辑，收到group命令后，输入内容发送给client
		/*******************************************************
			存在一个问题是，与服务器之间的通信时在default中，
			发送完group之后进入default中，等待下一个输入

			解决办法：
		******************************************************/
		select {
		case groupInfo := <-groupCh:
			{
				if groupInfo != "quit" {
					//收到回话要求
					groupList := strings.Split(groupInfo, "/")
					remoteUdpAddr, _ := net.ResolveUDPAddr("udp", groupList[1])
					remoteChatConn, err := net.DialUDP("udp", nil, remoteUdpAddr)
					if err != nil {
						logger.Panic(err)
						fmt.Println(err)
					}
					//fmt.Printf("chat with client:%s\n",groupList[2])
					encoder := json.NewEncoder(remoteChatConn)
					quitFlag := false
					var mess Message
					for {
						if quitFlag {
							break
						} else {
							select {
							case str := <-inputCh:
								{
									if str == "quit" {
										quitFlag = true
										mess = Message{
											Cmd:      "quit",
											Sender:   u.name,
											Data:     "",
											Receiver: groupList[0],
										}
									} else {
										mess = Message{
											Cmd:      "chat",
											Sender:   u.name,
											Data:     str,
											Receiver: groupList[0],
										}
									}
									err := encoder.Encode(mess)
									if err != nil {
										fmt.Println(err)
										logger.Panic(err)
									}
								}
							case groupInfo := <-groupCh:
								{
									if groupInfo == "quit" {
										//退出会话，同时向服务器反馈
										fmt.Printf("%s left the chatting\n", groupList[0])
										quitFlag = true
										remoteChatConn.Close()
										mess = Message{
											Cmd:      "quit",
											Sender:   u.name,
											Data:     groupList[0],
											Receiver: "server",
										}
										encoder := json.NewEncoder(chatConn)
										err := encoder.Encode(mess)
										if err != nil {
											fmt.Println(err)
											logger.Panic(err)
										}
									}
								}
							}
						}
					}
				}

			}
		case str := <-inputCh:
			{
				fmt.Println(str)
				if err != nil {
					fmt.Println(err)
					logger.Panic(err)
				}
				sendFlag := false
				if str == "list" {
					mess = Message{
						Cmd:      str,
						Sender:   u.name,
						Data:     "",
						Receiver: "server",
					}
					sendFlag = true
				} else if str == "quit" {
					mess = Message{
						Cmd:      str,
						Sender:   u.name,
						Data:     "",
						Receiver: "server",
					}
					sendFlag = true
				} else {
					lists := strings.Split(str, " ")
					if lists[0] == "group" {
						if len(lists) == 2 {
							if lists[1] == u.name { //如果选着跟自己交谈，就没必要了吧
								fmt.Println("you can talk to yourself without me")
							} else {
								mess = Message{
									Cmd:      lists[0],
									Sender:   u.name,
									Data:     strings.Join(lists[1:len(lists)], "/"),
									Receiver: "server",
								}
								sendFlag = true
							}
						} else { //暂时只有两人间的会话
							fmt.Println("just support conversation between 2 clients")
						}
					}
				}
				if sendFlag {
					encoder := json.NewEncoder(chatConn)
					err := encoder.Encode(mess)
					if err != nil {
						fmt.Println(err)
						logger.Panic(err)
					}
				}
			}
		}
	}
}
func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	listenPort := "172.16.18.163:8080"
	file, err := os.OpenFile("log.txt", os.O_APPEND, 0666)
	if err != nil {
		file, err = os.Create("log.txt")
		if err != nil {
			fmt.Println(err)
		}
	}
	year, month, day := time.Now().Date()
	logger := log.New(file, fmt.Sprintf("R:%v-%v-%v:", year, month, day), log.Ltime)
	temp := Login(listenPort)
	fmt.Println(temp)
	fmt.Println(temp.reader.LocalAddr().String())
	groupCh := make(chan string)
	go temp.Read(groupCh, logger)
	temp.Write(groupCh, logger)
}
