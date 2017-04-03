package tcp2

import (
	"encoding/binary"
	"fmt"
	"errors"
)
/**
	固定长度包头编解码器
 */
type ProtoFixedHead struct {
	readbuf []byte //数据读取缓冲BUF
	dl      uint32 //数据开始的位置
	rl      uint32 //数据结束的位置
	session *Session
}

const (
	PFH_HEAD_LEN            uint32 = 5 //包头长度
	PFH_HEAD_VERSION_POS    uint32 = 0  //版本字节的开始位置
	PFH_HEAD_PACKETLEN_POS  uint32 = 1  //消息长度字节的开始位置
)
//定长包头协议生成器
type FiexdHeadProtocolGenerator struct{

}
//产生协议对象
func (self * FiexdHeadProtocolGenerator) New(session *Session)(protocol Protocol){
	protocol = &ProtoFixedHead{
		readbuf: make([]byte, RECV_BUF),
		session: session,
	}
	return
}

//协议头
type _PFHPacketHead struct {
	Version  byte   //版本号    1字节
	Datalen  uint32 //实体长度   4字节
}

//判断是否需要移动剩余缓冲字节到buf的开始位置
func (self *ProtoFixedHead) checkReadBuffer() error {
	if (RECV_BUF - self.rl) < MIN_BUF {
		copy(self.readbuf, self.readbuf[self.dl:self.rl])
		self.rl = self.rl - self.dl
		self.dl = 0
	}
	if (self.rl - self.dl) > BYTE_MAX_BUF {
		fmt.Errorf("包过大,包长：%d", self.rl-self.dl)
		return TOO_LAGER
	}
	return nil
}

//读取数据
func (self *ProtoFixedHead) read() error{
	conn := self.session.conn
	//clientAddr := conn.RemoteAddr()
	//count := 0
	readbuf := self.readbuf
	for {
		if err := self.checkReadBuffer(); err != nil {
			self.session.Close()
			self.session.onError(err)
			self.session.onClose()
			return err
		}
		n, err := conn.Read(readbuf[self.rl:])
		if err != nil {
			if !self.session.isClosed {
				self.session.Close()
				if err.Error() == "EOF" {} else {
					self.session.onError(err)
				}
				self.session.onClose()
			}
			return err
		}
		if n <= 0 {
			if !self.session.isClosed {
				self.session.Close()
				self.session.onClose()
			}
			return err
		}
		self.rl += uint32(n)
		//count += n
		//logger.Debug("接收到数据长度:", n)
		//fmt.Println("接收到时数据len：", n)
		err = self.splitPackage(readbuf)
		if err != nil{
			if !self.session.isClosed {
				self.session.Close()
				self.session.onError(err)
				self.session.onClose()
			}
			return err
		}
	}
}

//拆包
func (self *ProtoFixedHead) splitPackage(readbuf []byte) (err error){
	//logger.Debug("数据:", readbuf[self.dl:self.rl])
	for {
		//当前消息包的长度
		if self.rl-self.dl < PFH_HEAD_LEN {
			break
		}
		packlen := binary.BigEndian.Uint32(readbuf[self.dl + PFH_HEAD_PACKETLEN_POS:])
		if packlen > BYTE_MAX_BUF{
			err = TOO_LAGER
			break
		}

		//计算出 完整包的结束游标
		completelen := self.dl + PFH_HEAD_LEN + uint32(packlen)
		if self.rl < completelen {
			break
		}
		packet := readbuf[self.dl:completelen]
		self.dl = completelen
		self.session.onData(packet[PFH_HEAD_LEN:])
	}
	return
}

func (self *ProtoFixedHead) write(data []byte,params ...interface{})(int,error){
	if data == nil{
		return 0,errors.New("Send nil data")
	}
	dataLen:=uint32(len(data))
	head := _PFHPacketHead{Version:1,Datalen:dataLen}
	rdata := head.ToByte()
	if dataLen > 0 {
		copy(rdata[PFH_HEAD_LEN:], data)
	}
	//fmt.Println("write data:",rdata)
	return self.session.conn.Write(rdata)
}


func (self *_PFHPacketHead) ToByte() []byte {
	data := make([]byte, self.Datalen+uint32(PFH_HEAD_LEN))
	data[HEAD_VERSION_POS] = self.Version
	binary.BigEndian.PutUint32(data[PFH_HEAD_PACKETLEN_POS:], self.Datalen)
	return data
}
func (self *_PFHPacketHead) ToString() {
	fmt.Println("版本号：", self.Version,  "包长度：", self.Datalen)
}
