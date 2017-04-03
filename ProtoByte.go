package tcp2

import (
	"encoding/binary"
	"fmt"
)
/**
	固定长度包头编解码器，包头里面包含接收目标地址，适合IM做透明投递，不需要解码消息体
 */
const (
	RECV_BUF uint32 = 100 * 1024
	BYTE_MAX_BUF  uint32 = 1024 * 1024 * 10  //10M
	MIN_BUF  uint32 = 10 * 1024
)

type ProtoByte struct {
	readbuf []byte //数据读取缓冲BUF
	dl      uint32 //数据开始的位置
	rl      uint32 //数据结束的位置
	session *Session
}

const (
	HEAD_LEN           uint32 = 12 //包头长度
	HEAD_VERSION_POS   uint32 = 0  //版本字节的开始位置
	HEAD_MSGTYPE_POS   uint32 = 1  //消息类型字节的开始位置
	HEAD_PACKETLEN_POS uint32 = 2  //消息长度字节的开始位置
	HEAD_TARGETID_POS  uint32 = 4  //目标字节的开始位置
)
//定长包头协议生成器
type ByteProtocolGenerator struct{}
//产生协议对象
func (self * ByteProtocolGenerator) New(object interface{})(protocol Protocol){
	protocol= &ProtoByte{
		readbuf: make([]byte, RECV_BUF),
		session: object.(*Session),
	}
	return
}

//协议头
type PacketHead struct {
	Version  byte   //版本号       1字节
	Msgtype  byte   //消息类型   1字节
	Datalen  uint16 //实体长度   2字节
	Targetid uint64 //目标ID   8字节
}

//判断是否需要移动剩余缓冲字节到buf的开始位置
func (self *ProtoByte) checkReadBuffer() error {
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
func (self *ProtoByte) read() error{
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
				if err.Error() != "EOF" {
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
		//fmt.Println("接收到时数据：", n, " 总数据：", count, "时间：", time.Since(now))
		self.splitPackage(readbuf)
	}
}

//拆包
func (self *ProtoByte) splitPackage(readbuf []byte) (err error){
	//logger.Debug("数据:", readbuf[self.dl:self.rl])
	for {
		//当前消息包的长度
		if self.rl-self.dl < HEAD_LEN {
			break
		}
		packlen := binary.BigEndian.Uint16(readbuf[self.dl+HEAD_PACKETLEN_POS:])
		//计算出 完整包的结束游标
		completelen := self.dl + HEAD_LEN + uint32(packlen)
		if self.rl < completelen {
			break
		}
		packet := readbuf[self.dl:completelen]
		self.dl = completelen
		self.session.onData(packet)
	}
	return
}
func (self *ProtoByte) write(data []byte,params ...interface{})(int,error){
	msgtype :=params[0].(byte)
	dl := 0
	if data != nil {
		dl = len(data)
	}
	ph := &PacketHead{Version:1,Msgtype:msgtype,Datalen:uint16(dl),Targetid:0}
	rdata := ph.ToByte()
	if data != nil && dl > 0 {
		copy(rdata[HEAD_LEN:], data)
	}

	return self.session.conn.Write(rdata)
}

/*func NewPacketHead(data []byte) (ph *PacketHead) {
	packlen := binary.BigEndian.Uint16(data[HEAD_PACKETLEN_POS:])
	ph = NewPacketHead2(data[HEAD_VERSION_POS], data[HEAD_MSGTYPE_POS], packlen, binary.BigEndian.Uint64(data[HEAD_TARGETID_POS:]))
	return
}*/

func (self *PacketHead) ToByte() []byte {
	data := make([]byte, self.Datalen+uint16(HEAD_LEN))
	data[HEAD_VERSION_POS] = self.Version
	data[HEAD_MSGTYPE_POS] = self.Msgtype
	binary.BigEndian.PutUint16(data[HEAD_PACKETLEN_POS:], self.Datalen)
	binary.BigEndian.PutUint64(data[HEAD_TARGETID_POS:], self.Targetid)
	return data
}
func (self *PacketHead) ToString() {
	fmt.Println("版本号：", self.Version, "消息类型：", self.Msgtype, "包长度：", self.Datalen, "目标ID：", self.Targetid)
}
