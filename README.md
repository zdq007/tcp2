# tcp2
golang网络通讯第二版

# 引入
go get github.com/zdq007/tcp2

# 使用例子：
# server
    import (
        "fmt"
        "github.com/zdq007/tcp2"
        "os"
    )
    
    func main() {
        server := tcp2.CreateServer(OnConnect)
        err := server.Listen("*:9090", func(err error, server *tcp2.Server) {
            if err != nil {
                fmt.Println(err)
                os.Exit(0)
            }
            server.SetProtocol(&tcp2.FiexdHeadProtocolGenerator{})
            server.SetHeartTime(60)
            fmt.Println("server started")

        })

        fmt.Println("server stoped by :", err)
    }

    //当链接进来时触发
    func OnConnect(session *tcp2.Session) {
        //session.SetTimeout(10)  链接进来设置超时，规定时间内不认关闭掉链接
        //session.ClearTimeout() 认证过后就清除掉超时

        /*go func() {
            for{
                time.Sleep(time.Second*1)
                session.Send([]byte("test"))
            }
        }()*/
        fmt.Println("onconnect")
        // 开启心跳检测
        session.On("data", tcp2.OnData(func(data []byte) {
            fmt.Println("recive data:", string(data))
            if string(data) == "" {
                session.Heart()
                session.Send([]byte(""))
            }

        }))
        session.On("error", tcp2.OnError(func(err error) {
            fmt.Println("connect error:", err)
            session.ClearHeart()

        }))
        session.On("close", tcp2.OnClose(func() {
            session.Close()
            fmt.Println("connect close")
            session.ClearHeart()

        }))
    }
