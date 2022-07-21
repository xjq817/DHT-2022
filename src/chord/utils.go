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
)

var (
	two = big.NewInt(2)
	mod = new(big.Int).Exp(two,big.NewInt(int64(M)),nil)
)

func myAccpet(server *rpc.Server,listener ner.Listener,n *ChordNode) {
	for {
		conn,err:=listener.Accept()
		select {
		case <-n.quitSignal:
			return
		default:
			if err!=nil{
				return
			}
			go server.ServeComm(conn)
		}
	}
}

func id(addr string) *big.Int {
	h:=sha1.New()
	h.Write([]byte(addr))
	return (&big.Int{}).SetBytes(h.Sum(nil))
}

func closeClient(client *rpc.Client) {
	err:=client.Close()
}

func Dial(addr string) (*rpc.Client,error) {
	if addr==""{
		return nil,errors.New("dial a null addr")
	}
	var client *rpc.Client
	var err error
	errChan:=make(chan error)
	for i:=0;i<5;i++{
		go func(){
			client,err=rpc.Dial("tpc",addr)
			errChan<-err
		}()
		select{
		case <-errChan:
			if err==nil{
				return client,nil
			}
			else{
				return nil,err
			}
		case <-time.After(dialTime):
			err=errors.New(fmt.Sprintln("dail TLE",addr))
		}
	}
	return nil,err
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
	var client *rpc.Client
	var err error
	errChan:=make(chan error)
	for i:=0;i<5;i++{
		go func(){
			client,err=rpc.Dial("tpc",addr)
			errChan<-err
		}()
		select{
		case <-errChan:
			if err==nil{
				return client,nil
			}
			else{
				return nil,err
			}
		case <-time.After(dialTime):
			err=errors.New(fmt.Sprintln("dail TLE",addr))
		}
	}
}