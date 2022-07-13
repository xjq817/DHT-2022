package chord

import(
	"errors"
	log "github.com/sirupsen/logrus"
	"math/big"
	"net"
	"net/rpc"
	"sync"
	"time"
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
}

func (n *ChordNode) initialize(addr string) {
	n.addr=addr
	n.store=make(map[string]string)
	n.preStore=make(map[string]string)
}

func (n *ChordNode) findSuccessor(request *big.Int,reply *string) error {
	var suc string

}

func (n *ChordNode) findPredecessor(request string,reply *string) error {}

func (n *ChordNode) closestPrecedingFinger(request *big.Int) (string,error) {}

func (n *ChordNode) join(addr string) bool {}

func (n *ChordNode) stabilize() {}

func (n *ChordNode) notify(request string,reply *string) error {}

func (n *ChordNode) fixFingers() {}

func (n *ChordNode) maintain() {
	go func(){
		for {
			if n.online {
				n.stabilize()
			}
			time.Sleep(maintainPauseTime)
		}
	}()
	go func(){
		for {
			if n.online {
				n.fixFingers()
			}
			time.Sleep(maintainPauseTime)
		}
	}()
}