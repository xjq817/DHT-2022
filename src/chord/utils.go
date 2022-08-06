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
	M                = 160
	successorListLen = 5
	maintainTime     = 200 * time.Millisecond
	dialTime         = 500 * time.Millisecond
	pingTime         = 500 * time.Millisecond
)

var (
	two = big.NewInt(2)
	mod = new(big.Int).Exp(two, big.NewInt(int64(M)), nil)
)

type Pair struct {
	Key   string
	Value string
}

func MyAccept(server *rpc.Server, listener net.Listener, n *ChordNode) {
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

func whereMod(nId *big.Int, i int) *big.Int {
	u := new(big.Int).Exp(two, big.NewInt(int64(i)), nil)
	u = new(big.Int).Add(nId, u)
	return new(big.Int).Mod(u, mod)
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

func isIn(key, start, end *big.Int, endStatus bool) bool {
	if start.Cmp(end) < 0 {
		if start.Cmp(key) >= 0 {
			return false
		}
		if endStatus {
			return key.Cmp(end) <= 0
		} else {
			return key.Cmp(end) < 0
		}
	} else {
		if start.Cmp(key) < 0 {
			return true
		}
		if endStatus {
			return key.Cmp(end) <= 0
		} else {
			return key.Cmp(end) < 0
		}
	}
}

// func logErrorFunctionCall(addr, fromFunc, toFunc string, err error) {
// 	log.Errorf("[Addr:%v] In call from [%v] to [%v] failed, error message: [%v].", addr, fromFunc, toFunc, err)
// }
