package tcp2

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
	"fmt"
)

/**
 *@mark+ 	tcp服务器封装，简化了tcp的调用流程，并封装了简单的协议buf，实现了数据流的拆分和封装
 *@author 	zdq
 *@time  	2015-8-12
 */
type Server struct {
	listener          *net.TCPListener
	htw 			  *HeartTimeWheel
	sessions          []*Session
	protocolGenerator ProtocolGenerator
	onConnect         OnConnect
}

var (
	addrFormatError = errors.New("Address format error, you should enter this type of address (ip:port).")
)

type Listener func(err error, server *Server)

/**
 *@mark+   创建服务
 *@param  onConnect 连接处理函数
 *@return 服务对象
 */
func CreateServer(onConnect OnConnect) *Server {
	var protocolGenerator ProtocolGenerator
	protocolGenerator = &DefaulJsonProtocolGenerator{}
	return NewServer(protocolGenerator, onConnect)
}

/**
 *@mark+   创建服务
 *@param  protocol  协议
 *@param  onConnect 连接处理函数
 *@return 服务对象
 */
func NewServer(protocolGenerator ProtocolGenerator, onConnect OnConnect) *Server {
	if protocolGenerator == nil {
		protocolGenerator = &DefaulJsonProtocolGenerator{}
	}
	if onConnect == nil {
		onConnect = func(session *Session) {}
	}
	return &Server{protocolGenerator: protocolGenerator, onConnect: onConnect}
}

func (self *Server) SetProtocol(protocolGenerator ProtocolGenerator) error {
	if protocolGenerator == nil{
		return errors.New("Nil protocol")
	}

	self.protocolGenerator = protocolGenerator
	return nil
}

/**
 *@mark+   设置心跳时间，并开启心跳检测功能，需要由上层配合，上层收到session收到心跳包后调用 session.Heart()
 *@param  timeSec 心跳超时时间
 */
func (self *Server) SetHeartTime(timeSec int)  {
	self.htw = createHeartTimeWheel(timeSec)
	go self.htw.start()
}

/**
 *@mark+   开启服务监听
 *@param  addr 		连接地址，示例：127.0.0.1:80  *:80 :80
 *@param  listener 	监听回调函数
 *@return 错误对象
 */
func (self *Server) Listen(addr string, listener Listener) error {
	ipAndPort := strings.Split(addr, ":")
	ipstr := "*"
	var port int
	if len(ipAndPort) < 2 {
		return addrFormatError
	} else {
		var err error
		if len(ipAndPort[0]) > 0 {
			ipstr = ipAndPort[0]
		}
		port, err = strconv.Atoi(ipAndPort[1])
		if err != nil {
			return addrFormatError
		}
	}
	paddr := &net.TCPAddr{IP: net.ParseIP(ipstr), Port: port}
	l, err := net.ListenTCP("tcp", paddr)
	listener(err, self)
	if err == nil {
		self.listener = l
		return self.accept()
	} else {
		return err
	}
}

/**
 *@mark-  循环堵塞接收连接
 */
func (self *Server) accept() error {
	for {
		conn, err := self.listener.AcceptTCP()
		if err != nil {
			return err
		}
		session := NewSession(self.protocolGenerator, conn)
		session.htw = self.htw
		self.onConnect(session)
		go session.readLoop()
	}
}
/**
 *@mark  心跳时间轮
 */
type HeartTimeWheel struct {
	wheel  []map[string]*Session
	index  int
	size   int
	runing bool
	mutex  sync.Mutex
}

/**
 *@mark+  		创建心跳时间轮
 *@param  size  指定时间轮长度，即多少秒超时
 */
func createHeartTimeWheel(size int) (htw *HeartTimeWheel) {
	htw = new(HeartTimeWheel)
	htw.size = size

	htw.wheel = make([]map[string]*Session, size, size)
	for i := range htw.wheel {
		htw.wheel[i] = make(map[string]*Session)
	}
	return
}

func (self *HeartTimeWheel) start() {
	go self.timer()
}
func (self *HeartTimeWheel) timer() {
	self.runing = true
	for self.runing {
		time.Sleep(time.Second * 1)
		self.check()
	}
}
func (self *HeartTimeWheel) check() {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	self.index++
	self.index = self.index%self.size
	//fmt.Println(self.index)
	timeOutObjs := self.wheel[self.index]
	size := len(timeOutObjs)
	if size > 0 {
		self.wheel[self.index] = make(map[string]*Session)
		//处理超时的session
		go self.handleHeartTimeout(timeOutObjs)
	}
}
func (self *HeartTimeWheel) handleHeartTimeout(sessions map[string]*Session){
	fmt.Println("超时session :",sessions)
	for _,session:=range sessions{
		session.Close()
		session.onClose()
	}
}
const HEART_WHEEL_POS = "HEART_WHEEL_POS"
func (self *HeartTimeWheel) in( session *Session) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if session.IsSet(HEART_WHEEL_POS){
		oldindex := session.GetInt(HEART_WHEEL_POS)
		delete(self.wheel[oldindex],session.sid)
	}
	session.Set(HEART_WHEEL_POS,self.index)
	self.wheel[self.index][session.sid] = session

}

func (self *HeartTimeWheel) out(session *Session)  {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if session.IsSet(HEART_WHEEL_POS){
		oldindex := session.GetInt(HEART_WHEEL_POS)
		delete(self.wheel[oldindex],session.sid)
	}
}