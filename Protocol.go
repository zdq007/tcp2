package tcp2

import(
	"errors"
)
//包过大
var TOO_LAGER = errors.New("Package is to lagger!")
var PROTO_ERR = errors.New("Proto format is err!")

//协议生成器接口
type ProtocolGenerator interface{
	New(*Session) Protocol
}
//默认协议生成器
type DefaulJsonProtocolGenerator struct{
	
}
//产生协议对象
func (self * DefaulJsonProtocolGenerator) New(session *Session)(protocol Protocol){
	protocol=NewProtoJson(session)
	return
}

//协议接口,协议负责拆包和封包
type Protocol interface {
	read() error
	write([]byte,...interface{})(int,error)
	splitPackage([]byte) error
}
