package kademlia

import (
	"crypto/sha1"
	"errors"
	"math/big"
	"net"
	"net/rpc"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	M             = 160
	K             = 20
	alpha         = 3
	dialTime      = 500 * time.Millisecond
	pingTime      = 500 * time.Millisecond
	durationTime  = 500 * time.Millisecond
	maintainTime  = 2 * time.Second
	RepublishTime = 120 * time.Second
	ExpireTime    = 960 * time.Second
)

type DataPair struct {
	Key   string
	Value string
}

type FindValuePair struct {
	List  ClosestList
	Value string
}

type StorePair struct {
	Value DataPair
	Addr  string
}

type ConcurrencyType struct {
	List ClosestList
	Cnt  int
}

func logErrorFunctionCall(addr, fromFunc, toFunc string, err error) {
	log.Errorf("[Addr:%v] In call from [%v] to [%v] failed, error message: [%v].", addr, fromFunc, toFunc, err)
}

func MyAccept(server *rpc.Server, listener net.Listener, n *KadNode) {
	for {
		conn, err := listener.Accept()
		select {
		case <-n.quitSignal:
			return
		default:
			if err != nil {
				return
			}
			go server.ServeConn(conn)
		}
	}
}

func id(addr string) *big.Int {
	h := sha1.New()
	h.Write([]byte(addr))
	return (&big.Int{}).SetBytes(h.Sum(nil))
}

func dis(u *big.Int, v *big.Int) *big.Int {
	return new(big.Int).Xor(u, v)
}

func cpl(u *big.Int, v *big.Int) int {
	return (*dis(u, v)).BitLen() - 1
}

func Dial(addr string) (*rpc.Client, error) {
	if addr == "" {
		return nil, errors.New("dial a null addr")
	}
	var client *rpc.Client
	var err error
	errChan := make(chan error)
	for i := 0; i < 5; i++ {
		go func() {
			client, err = rpc.Dial("tcp", addr)
			errChan <- err
		}()
		select {
		case <-errChan:
			if err == nil {
				return client, nil
			} else {
				return nil, err
			}
		case <-time.After(dialTime):
		}
	}
	return nil, err
}

func RemoteCall(addr string, funcName string, request interface{}, reply interface{}) error {
	client, err := Dial(addr)
	if err != nil {
		return err
	}
	err = client.Call(funcName, request, reply)
	_ = client.Close()
	return err
}

func Ping(addr string) bool {
	if addr == "" {
		return false
	}
	errChan := make(chan error)
	for i := 0; i < 5; i++ {
		go func() {
			client, err := rpc.Dial("tcp", addr)
			if err == nil {
				_ = client.Close()
			}
			errChan <- err
		}()
		select {
		case err := <-errChan:
			if err == nil {
				return true
			} else {
				return false
			}
		case <-time.After(pingTime):
		}
	}
	return false
}
