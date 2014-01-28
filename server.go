package main

import (
	"encoding/json"
	"fmt"
	"log"
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
*@param Name：用户名
*@param BeatCount：心跳累计
*****************************************************/
type User struct {
	Name       string
	Addr       string
	RemoteName string
	BeatCount  int
}

/****************************************************
*@brief 定义在线用户存储结构
*****************************************************
*@param shelf：存储结构
*****************************************************/
type Store struct {
	shelf map[string]User
}

/****************************************************
*@function (s *Store) Sort(logger *log.Logger)
*****************************************************
*@brief 定时对在线用户进行心跳检查，
*****************************************************
*@access Public
*****************************************************
*@param logger *log.Logger:日志文件
*****************************************************
*@return 无
*****************************************************/
func (s *Store) Sort(logger *log.Logger) {
	timer := time.NewTimer(2 * time.Second)
	for {
		select {
		case <-timer.C:
			{
				logger.Printf("online users:%v", s.shelf)
				for tempName, tempUser := range s.shelf {
					if tempName != "server" {
						if tempUser.BeatCount <= 1 {
							if tempUser.RemoteName != "server" {
								remoteAddr, err := net.ResolveUDPAddr("udp", s.shelf[tempUser.RemoteName].Addr)
								if err != nil {
									fmt.Println(err)
									logger.Printf("sort:%v", err)
								}
								quitConn, err := net.DialUDP("udp", nil, remoteAddr)
								if err != nil {
									fmt.Println(err)
									logger.Printf("sort:%v", err)
								}
								defer quitConn.Close()
								encoder := json.NewEncoder(quitConn)
								mess := Message{
									Cmd:      "quit",
									Sender:   "server",
									Data:     "",
									Receiver: tempUser.RemoteName,
								}
								err = encoder.Encode(mess)
								if err != nil {
									fmt.Println(err)
									logger.Printf("sort:%v", err)
								}
							}
							s.Delete(tempName)
							tempUser = s.shelf[tempUser.RemoteName]
							s.shelf[tempUser.RemoteName] = User{
								Name:       tempUser.Name,
								Addr:       tempUser.Addr,
								RemoteName: "server",
								BeatCount:  tempUser.BeatCount,
							}
						} else {
							s.shelf[tempName] = User{
								Name:       tempUser.Name,
								Addr:       tempUser.Addr,
								RemoteName: tempUser.RemoteName,
								BeatCount:  0,
							}
						}
					}
				}
			}
		}
		timer.Reset(3 * time.Second)
	}
}

/****************************************************
*@function func (s *Store) Add(name string, user User)
*****************************************************
*@brief 添加到在线用户组
*****************************************************
*@access Public
*****************************************************
*@param name：用户名
*@param user:详细用户信息
*****************************************************
*@return 无
*****************************************************/
func (s *Store) Add(name string, user User) {
	s.shelf[name] = user
}

/****************************************************
*@function func (s *Store) Change(name string, user User)
*****************************************************
*@brief 修改在线用户组中的某个用户
*****************************************************
*@access Public
*****************************************************
*@param name：用户名
*@param user:详细用户信息
*****************************************************
*@return 无
*****************************************************/
func (s *Store) Change(name string, user User) {
	_, flag := s.shelf[name]
	if flag {
		s.shelf[name] = user
	}
}

/****************************************************
*@function func (s *Store) Delete(name string, user User)
*****************************************************
*@brief 删除在线用户组中的某个用户
*****************************************************
*@access Public
*****************************************************
*@param name：用户名
*****************************************************
*@return 无
*****************************************************/
func (s *Store) Delete(name string) {
	delete(s.shelf, name)
}

/****************************************************
*@function func (s *Store) Beat(name string, user User)
*****************************************************
*@brief 在线用户组中的某个用户接收心跳
*****************************************************
*@access Public
*****************************************************
*@param name：用户名
*****************************************************
*@return 无
*****************************************************/
func (s *Store) Beat(name string) {
	tempCell, flag := s.shelf[name]
	if flag == true {
		s.shelf[name] = User{
			Name:       tempCell.Name,
			Addr:       tempCell.Addr,
			RemoteName: tempCell.RemoteName,
			BeatCount:  tempCell.BeatCount + 1,
		}
	}
}

/****************************************************
*@function func (s *Store) GetUser(name string) User
*****************************************************
*@brief 输出在线用户组中的某个用户
*****************************************************
*@access Public
*****************************************************
*@param name：用户名
*****************************************************
*@return User：用户详细信息
*****************************************************/
func (s *Store) GetUser(name string) User {
	return s.shelf[name]
}

/****************************************************
*@function func (s *Store) GetMap()
*****************************************************
*@brief 输出在线用户组
*****************************************************
*@access Public
*****************************************************
*@param name：无
*****************************************************
*@return User：在线用户组
*****************************************************/
func (s *Store) GetMap() map[string]User {
	return s.shelf
}

/****************************************************
*@function NewStore (s *Store) Store
*****************************************************
*@brief 新建一个在线用户组
*****************************************************
*@access Public
*****************************************************
*@param name：无
*****************************************************
*@return User：在线用户组
*****************************************************/
func NewStore() Store {
	data := make(map[string]User)
	temp := new(Store)
	temp.shelf = data
	//加上对仓库的定时整理
	file, err := os.OpenFile("onlineusers.txt", os.O_APPEND, 0666)
	if err != nil {
		file, err = os.Create("onlineusers.txt")
		if err != nil {
			fmt.Println(err)
		}
	}

	year, month, day := time.Now().Date()
	logger := log.New(file, fmt.Sprintf("R:%v-%v-%v:", year, month, day), log.Ltime)
	go temp.Sort(logger)
	return *temp
}

/****************************************************
*@function Login(loginPort, listenPort,beatPort string, userCh chan User)
*****************************************************
*@brief 服务器开启登录服务，获取user远程端口、发送welcome
*		介绍，返回已登录用户的信息
		向chat监听端口发送用户信息，保证登录用户的同步性
*****************************************************
*@access Public
*****************************************************
*@param loginPort string 登陆认证端口号
*@param listenPort string 监听端口号
*@param userCh chan User 用户类型channel
*****************************************************
*@return 无
*****************************************************/
func Login(loginPort, listenPort string, userCh chan User, logger *log.Logger) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	//创建map
	UserMap := make(map[string]User)
	UserMap["server"] = User{
		Name:       "server",
		Addr:       listenPort,
		RemoteName: "server",
		BeatCount:  2,
	}
	//开启登录监听端口
	loginService, err := net.Listen("tcp", loginPort)
	if err != nil {
		fmt.Println(err)
		logger.Printf("login:%v\n", err)
	}
	fmt.Println("login listener setted successfully")
	for {
		//接收tcp连接
		loginConn, err := loginService.Accept()
		if err != nil {
			fmt.Println(err)
			logger.Printf("login:%v\n", err)
		}
		defer loginConn.Close()
		//处理登录请求
		go func(conn net.Conn, ch chan User) {
			//关闭连接
			defer conn.Close()
			//声明用户、消息以及json解码编码接口
			var mess Message
			var user User
			decoder := json.NewDecoder(conn)
			encoder := json.NewEncoder(conn)
			//json解码，收到login请求
			err = decoder.Decode(&mess)
			if err != nil {
				fmt.Println(err)
				logger.Printf("login:%v\n", err)
			}
			fmt.Printf("CMD:%v,DATA:%v,Sender:%v,Receiver:%v\n", mess.Cmd, mess.Data, mess.Sender, mess.Receiver)
			user.Addr = mess.Data
			//发送chat端口
			mess = Message{
				Cmd:      "connect",
				Sender:   conn.LocalAddr().String(),
				Data:     listenPort,
				Receiver: conn.RemoteAddr().String(),
			}
			//fmt.Printf("CMD:%v,DATA:%v,Sender:%v,Receiver:%v\n", mess.Cmd, mess.Data, mess.Sender, mess.Receiver)
			err = encoder.Encode(mess)
			if err != nil {
				fmt.Println(err)
				logger.Printf("login:%v\n", err)
			}
			//发送welcome、介绍
			mess = Message{
				Cmd:      "login",
				Sender:   conn.LocalAddr().String(),
				Data:     "welcome to use this communication app,input your name to login\nplease do not use these words:\n1:list;  2:group;  3.quit\n",
				Receiver: conn.RemoteAddr().String(),
			}
			//fmt.Printf("CMD:%v,DATA:%v,Sender:%v,Receiver:%v\n", mess.Cmd, mess.Data, mess.Sender, mess.Receiver)
			err = encoder.Encode(mess)
			if err != nil {
				fmt.Println(err)
				logger.Printf("login:%v\n", err)
				conn.Close()
			}
			flag := true
			//如果一直输入不对，一直处于监听状态
			for flag {
				//获取客户端输入的用户名
				err = decoder.Decode(&mess)
				if err != nil {
					fmt.Println(err)
					logger.Printf("login:%v\n", err)
					conn.Close()
					break
				}
				fmt.Printf("CMD:%v,DATA:%v,Sender:%v,Receiver:%v\n", mess.Cmd, mess.Data, mess.Sender, mess.Receiver)
				//校验用户名是否被占用
				//如果被占用，提示重新输入
				if "" != UserMap[mess.Data].Name {
					mess = Message{
						Cmd:      "login",
						Sender:   conn.LocalAddr().String(),
						Data:     "fail",
						Receiver: conn.RemoteAddr().String(),
					}
				} else {
					user.Name = mess.Data
					user.RemoteName = "server"
					user.BeatCount = 2
					UserMap[user.Name] = user
					mess = Message{
						Cmd:      "login",
						Sender:   "server",
						Data:     "success",
						Receiver: user.Name,
					}
					flag = false
				}
				//fmt.Printf("CMD:%v,DATA:%v,Sender:%v,Receiver:%v\n", mes.Cmd, mes.Data, mes.Sender, mes.Receiver)
				err = encoder.Encode(mess)
				if err != nil {
					fmt.Println(err)
					logger.Printf("login:%v\n", err)
					conn.Close()
					break
				}
			}
			logger.Printf("user %v login\n", user)
			ch <- user
		}(loginConn, userCh)
	}
}

/****************************************************
*@function ListenMess(listenPort string, userCh chan User, logger *log.Logger)
*****************************************************
*@brief 开启本地消息监听
*****************************************************
*@access Public
*****************************************************
*@param listenPort：监听端口
*@param userCh：用户注册进程
*@param logger：日志文件
*****************************************************
*@return 无
*****************************************************/
func ListenMess(listenPort string, userCh chan User, logger *log.Logger) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	onLineUsers := NewStore()
	server := User{
		Name:       "server",
		Addr:       listenPort,
		RemoteName: "server",
		BeatCount:  2,
	}
	onLineUsers.Add("server", server)
	udpAddr, err := net.ResolveUDPAddr("udp", listenPort)
	if err != nil {
		fmt.Println(err)
		logger.Printf("ListenMess:%v\n", err)
	}
	fmt.Println("chat listener setted successfully")
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println(err)
		logger.Printf("ListenMess:%v\n", err)
	}
	decoder := json.NewDecoder(conn)
	for {
		select {
		case tempUser := <-userCh:
			{
				fmt.Println("-------------------new User from login-----------------")
				logger.Printf("ListenMess insert new user:%v\n", tempUser)
				onLineUsers.Add(tempUser.Name, tempUser)
			}
		default:
			{
				var mess Message
				err := decoder.Decode(&mess)
				if err != nil {
					fmt.Println(err)
					logger.Printf("ListenMess:%v\n", err)
					continue
				}
				if mess.Cmd != "beat" {
					fmt.Printf("CMD:%v,DATA:%v,Sender:%v,Receiver:%v\n", mess.Cmd, mess.Data, mess.Sender, mess.Receiver)
					logger.Printf("CMD:%v,DATA:%v,Sender:%v,Receiver:%v\n", mess.Cmd, mess.Data, mess.Sender, mess.Receiver)
				}
				switch mess.Cmd {
				case "beat":
					{
						onLineUsers.Beat(mess.Sender)
					}
				case "list":
					{
						tempUser := onLineUsers.GetUser(mess.Sender)
						fmt.Println(tempUser)
						remoteUdpAddr, err := net.ResolveUDPAddr("udp", tempUser.Addr)
						if err != nil {
							fmt.Println(err)
							logger.Printf("ListenMess:%v\n", err)
							continue
						}
						remoteUdpConn, err := net.DialUDP("udp", nil, remoteUdpAddr)
						defer remoteUdpConn.Close()
						if err != nil {
							fmt.Println(err)
							logger.Printf("ListenMess:%v\n", err)
							continue
						}
						fmt.Println(remoteUdpConn.RemoteAddr())
						encoder := json.NewEncoder(remoteUdpConn)
						strList := make([]string, 0)
						/*
							修改
						*/
						for tempName, _ := range onLineUsers.GetMap() {
							if tempName != "server" {
								tempList := append(strList, tempName)
								strList = tempList
							}
						}
						mess = Message{
							Cmd:      "list",
							Sender:   "server",
							Data:     strings.Join(strList[0:], "/"),
							Receiver: mess.Sender,
						}
						err = encoder.Encode(mess)
						if err != nil {
							fmt.Println(err)
							logger.Printf("ListenMess:%v\n", err)
							continue
						}
					}
				case "group":
					{
						//fmt.Printf("CMD:%v,DATA:%v,Sender:%v,Receiver:%v\n", mess.Cmd, mess.Data, mess.Sender, mess.Receiver)
						//读取会话的对方的信息
						if onLineUsers.GetUser(mess.Data).Name != "" {
							//被动方在线
							if onLineUsers.GetUser(mess.Data).RemoteName != "server" {
								//被动方已经建立group会话
								remoteUdpAddr, err := net.ResolveUDPAddr("udp", onLineUsers.GetUser(mess.Sender).Addr)
								if err != nil {
									fmt.Println(err)
									continue
								}
								remoteUdpConn, err := net.DialUDP("udp", nil, remoteUdpAddr)
								defer remoteUdpConn.Close()
								if err != nil {
									fmt.Println(err)
									logger.Printf("ListenMess:%v\n", err)
									continue
								}
								encoder := json.NewEncoder(remoteUdpConn)
								mess = Message{
									Cmd:      "group",
									Sender:   "server",
									Data:     fmt.Sprintf("the user <%s> is chatting with other people", mess.Data),
									Receiver: mess.Receiver,
								}
								err = encoder.Encode(mess)
								if err != nil {
									fmt.Println(err)
									logger.Printf("ListenMess:%v\n", err)
									continue
								}
							} else {
								//被动方未建立group会话，建立会话
								clientUdpAddrs := make([]string, 0)
								tempList := append(clientUdpAddrs, onLineUsers.GetUser(mess.Data).Name, onLineUsers.GetUser(mess.Data).Addr)
								fmt.Println(tempList)
								//修改二者的属性，对RemoteName做标记
								tempUser := User{
									Name:       onLineUsers.GetUser(mess.Data).Name,
									Addr:       onLineUsers.GetUser(mess.Data).Addr,
									RemoteName: mess.Sender,
									BeatCount:  onLineUsers.GetUser(mess.Data).BeatCount,
								}
								onLineUsers.Change(mess.Data, tempUser)
								tempUser = User{
									Name:       onLineUsers.GetUser(mess.Sender).Name,
									Addr:       onLineUsers.GetUser(mess.Sender).Addr,
									RemoteName: mess.Data,
									BeatCount:  onLineUsers.GetUser(mess.Sender).BeatCount,
								}
								onLineUsers.Change(mess.Sender, tempUser)
								//向被呼叫方，发送通知
								go func(remoteClientAddr string, sponsor string) {
									remoteUdpAddr, err := net.ResolveUDPAddr("udp", remoteClientAddr)
									if err != nil {
										fmt.Println(err)
										logger.Printf("ListenMess:%v\n", err)
									}
									remoteUdpConn, err := net.DialUDP("udp", nil, remoteUdpAddr)
									defer remoteUdpConn.Close()
									if err != nil {
										fmt.Println(err)
										logger.Printf("ListenMess:%v\n", err)
									}
									encoder := json.NewEncoder(remoteUdpConn)
									mess = Message{
										Cmd:      "group",
										Sender:   "server",
										Data:     fmt.Sprintf("0/%s/%s", sponsor, onLineUsers.GetUser(sponsor).Addr),
										Receiver: mess.Receiver,
									}
									err = encoder.Encode(mess)
									if err != nil {
										fmt.Println(err)
										logger.Printf("ListenMess:%v\n", err)
									}
								}(onLineUsers.GetUser(mess.Data).Addr, mess.Sender)
								//向发起方返回远程client地址
								remoteUdpAddr, err := net.ResolveUDPAddr("udp", onLineUsers.GetUser(mess.Sender).Addr)
								if err != nil {
									fmt.Println(err)
									continue
								}
								remoteUdpConn, err := net.DialUDP("udp", nil, remoteUdpAddr)
								defer remoteUdpConn.Close()
								if err != nil {
									fmt.Println(err)
									logger.Printf("ListenMess:%v\n", err)
									continue
								}
								encoder := json.NewEncoder(remoteUdpConn)
								mess = Message{
									Cmd:      "group",
									Sender:   "server",
									Data:     fmt.Sprintf("1/%s", strings.Join(tempList, "/")),
									Receiver: mess.Receiver,
								}
								err = encoder.Encode(mess)
								if err != nil {
									fmt.Println(err)
									logger.Printf("ListenMess:%v\n", err)
									continue
								}
							}
						} else {
							//被叫方不在线
							remoteUdpAddr, err := net.ResolveUDPAddr("udp", onLineUsers.GetUser(mess.Sender).Addr)
							if err != nil {
								fmt.Println(err)
								continue
							}
							remoteUdpConn, err := net.DialUDP("udp", nil, remoteUdpAddr)
							defer remoteUdpConn.Close()
							if err != nil {
								fmt.Println(err)
								logger.Printf("ListenMess:%v\n", err)
								continue
							}
							encoder := json.NewEncoder(remoteUdpConn)
							mess = Message{
								Cmd:      "group",
								Sender:   "server",
								Data:     fmt.Sprintf("the user <%s> is not online", mess.Data),
								Receiver: mess.Receiver,
							}
							err = encoder.Encode(mess)
							if err != nil {
								fmt.Println(err)
								logger.Printf("ListenMess:%v\n", err)
								continue
							}
						}
					}
				case "quit":
					{
						fmt.Printf("CMD:%v,DATA:%v,Sender:%v,Receiver:%v\n", mess.Cmd, mess.Data, mess.Sender, mess.Receiver)
						tempUser := User{
							Name:       onLineUsers.GetUser(mess.Sender).Name,
							Addr:       onLineUsers.GetUser(mess.Sender).Addr,
							RemoteName: "server",
							BeatCount:  onLineUsers.GetUser(mess.Sender).BeatCount,
						}
						onLineUsers.Change(mess.Sender, tempUser)
						tempUser = User{
							Name:       onLineUsers.GetUser(mess.Data).Name,
							Addr:       onLineUsers.GetUser(mess.Data).Addr,
							RemoteName: "server",
							BeatCount:  onLineUsers.GetUser(mess.Data).BeatCount,
						}
						onLineUsers.Change(mess.Data, tempUser)
					}
				}

			}

		}

	}
}
func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	file, err := os.OpenFile("log.txt", os.O_APPEND, 0666)
	if err != nil {
		file, err = os.Create("log.txt")
		if err != nil {
			fmt.Println(err)
		}
	}
	year, month, day := time.Now().Date()
	logger := log.New(file, fmt.Sprintf("R:%v-%v-%v:", year, month, day), log.Ltime)
	listenPort := "172.16.18.163:8081"
	loginPort := "172.16.18.163:8080"
	userCh := make(chan User)
	go Login(loginPort, listenPort, userCh, logger)
	ListenMess(listenPort, userCh, logger)
}
