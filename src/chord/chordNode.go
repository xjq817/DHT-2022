package chord

import(
	"errors"
	"math/big"
	"net"
	"net/rpc"
	// RPC规则：
		// 方法只能有两个可序列化的参数，
		// 其中第二个参数是指针类型，
		// 并且返回一个error类型，同时必须是公开的方法。
	"sync"
	"time"
	"fmt"
)

type ChordNode struct{
	addr              string

	predecessor       string
	predecessorLock   sync.RWMutex

	fingerTable       [M]string
	fingerTableLock   sync.RWMutex

	successorList     [successorListLen]string
	successorListLock sync.RWMutex

	online            bool
	onlineLock        sync.RWMutex

	store             map[string]string
	storeLock         sync.RWMutex

	preStore          map[string]string
	preStoreLock      sync.RWMutex

	server            *rpc.server
	listener          net.Listener

	quitSignal        chan bool
}

func (n *ChordNode) initialize(addr string) {
	n.addr=addr
	n.store=make(map[string]string)
	n.preStore=make(map[string]string)
	n.quitSignal=make(chan bool,2)
}

func (n *ChordNode) initializeServer() {
	n.server=rpc.NewServer()
	err:=n.server.Register(n)
	if err!=nil{
		return
	}
	n.listener,err=net.Listen("tcp",n.addr)
	if err!=nil{
		return
	}
	go myAccept(n.server,n.listener,n)
}

//初始化
func (n *ChordNode) run() {
	n.initializeServer()
	n.maintain()
}

//加入第一个节点
func (n *ChordNode) create() {
	n.onlineLock.Lock()
	n.online=true
	n.onlineLock.Unlock()

	n.successorListLock.Lock()
	n.successorList[0]=n.addr
	n.successorListLock.Unlock()

	n.fingerTableLock.Lock()
	for i:=0;i<M;i++ {
		n.fingerTable[i]=n.addr
	}
	n.fingerTableLock.Unlock()

	n.predecessorLock.Lock()
	n.predecessor=n.addr
	n.predecessorLock.Unlock()
}

func getSuccessorList

//加入一个点
func (n *ChordNode) join(addr string) bool {
	if n.online{
		return false
	}

	n.predecessorLock.Lock()
	n.predecessor=""
	n.predecessorLock.Unlock()

	var suc string
	err := RemoteCall(n.addr,"chordNode.findSuccessor",id(n.addr),&suc)
	if err!=nil {
		return false
	}
	var sucList [successorListLen]string
	err = RemoteCall(suc,"chordNode.getSuccessorList","",&sucList)
	n.successorListLock.Lock()
	if err!=nil {
		return false
	}
	n.successorList[0]=suc
	for i:=1;i<successorListLen;i++ {
		n.successorList[i]=sucList[i-1]
	}
	n.successorListLock.Unlock()
	if suc!=n.addr {
		n.storeLock.Lock()
		err=RemoteCall(suc,"chordNode.transferData",n.addr,&n.store)
		///transferData
		n.storeLock.Unlock()
		if err!=nil {
			return false
		}
	}
	n.onlineLock.Lock()
	n.online=true
	n.onlineLock.Unlock()
	n.fingerTableLock.Lock()
	n.fingerTable[0]=suc
	n.fingerTableLock.Unlock()
	nId:=id(n.addr)
	for i:=1;i<M;i++ {
		var fingerNodeIth string
		err=remoteCall(suc,"chordNode.findSuccessor",whereMod(nId,i),&fingerNodeIth)
		////findSuccessor
		if err!=nil {
			fingerNodeIth=""
		}
		n.fingerTableLock.Lock()
		n.fingerTable[i]=fingerNodeIth
		n.fingerTableLock.Unlock()
	}
	return true
}

func (n *ChordNode) findSuccessor(request *big.Int,reply *string) error {}

func (n *ChordNode) findPredecessor(request string,reply *string) error {}

func (n *ChordNode) closestPrecedingFinger(request *big.Int) (string,error) {}

func (n *ChordNode) stabilize() {}

func (n *ChordNode) notify(request string,reply *string) error {}

func (n *ChordNode) fixFingers() {}

func (n *ChordNode) maintain() {
	go func() {
		for {
			if n.online {
				n.stabilize()
			}
			time.Sleep(maintainTime)
		}
	}()
	go func() {
		for {
			if n.online {
				n.fixFingers()
			}
			time.Sleep(maintainTime)
		}
	}()
}

func (n *ChordNode) quit() {}

func (n *ChordNode) forceQuit() {}

func (n *ChordNode) ping(addr string) bool {}

func (n *ChordNode) put(key string,value string) bool {}

func (n *ChordNode) get(key string) (bool,string) {}

func (n *ChordNode) delete(key string) bool {}