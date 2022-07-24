package chord

import (
	"crypto/sha1"
	"errors"
	"math/big"
	"net"
	"net/rpc"
	"time"
)

const (
	M                 = 160
	successorListLen  = 5
	maintainTime      = 200 * time.Millisecond
	dialTime          = 500 * time.Millisecond
	pingTime          = 500 * time.Millisecond
)

var (
	two = big.NewInt(2)
	mod = new(big.Int).Exp(two,big.NewInt(int64(M)),nil)
)

type Pair struct {
	Key   string
	Value string
}

func myAccept(server *rpc.Server,listener net.Listener,n *ChordNode) {
	for {
		conn,err:=listener.Accept()
		select {
		case <-n.quitSignal:
			return
		default:
			if err!=nil{
				return
			}
			go server.ServeConn(conn)
		}
	}
}

func id(addr string) *big.Int {
	h:=sha1.New()
	h.Write([]byte(addr))
	val:=new(big.Int)
	val.SetBytes(h.Sum(nil))
	return val
}

func closeClient(client *rpc.Client) {
	_=client.Close()
}

func Dial(addr string) (*rpc.Client,error) {
	if addr=="" {
		return nil,errors.New("dial a null addr")
	}
	var client *rpc.Client
	errChan:=make(chan error)
	for i:=0;i<5;i++{
		go func(){
			var err error
			client,err=rpc.Dial("tpc",addr)
			errChan<-err
		}()
		select{
		case err:=<-errChan:
			if err==nil {
				return client,nil
			} else {
				return nil,err
			}
		case <-time.After(dialTime):
			_=errors.New("dial more")
		}
	}
	return nil,errors.New("dail TLE")
}

func remoteCall(addr string,funcName string,request interface{},reply interface{}) error {
	client,err:=Dial(addr)
	if err!=nil{
		return err
	}
	defer closeClient(client)
	err=client.Call(funcName,request,reply)
	return err
}

func pow2(i int) *big.Int {
	return new(big.Int).Exp(two,big.NewInt(int64(i)),nil)
}

func whereMod(nId *big.Int,i int) *big.Int {
	return new(big.Int).Mod(new(big.Int).Add(nId,pow2(i)),mod)
}

func Ping(addr string) bool {
	if addr=="" {
		return false
	}
	errChan:=make(chan error)
	for i:=0;i<5;i++{
		go func(){
			client,err:=rpc.Dial("tpc",addr)
			if err==nil {
				_=client.Close()
			}
			errChan<-err
		}()
		select{
		case err:=<-errChan:
			if err==nil{
				return true
			} else {
				return false
			}
		case <-time.After(pingTime):
			_=errors.New("ping more")
		}
	}
	return false
}

func isIn(key,start,end *big.Int,endStatus bool) bool {
	if start.Cmp(end)<0 {
		if start.Cmp(key)>=0 {
			return false
		}
		if endStatus {
			return key.Cmp(end)<=0
		} else {
			return key.Cmp(end)<0
		}
	} else {
		if start.Cmp(key)<0{
			return true
		}
		if endStatus {
			return key.Cmp(end)<=0
		} else {
			return key.Cmp(end)<0
		}
	}
}