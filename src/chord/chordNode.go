package chord

import(
	"sync"
)

type ChordNode struct{
	addr              string
	predecessor       string
	predecessorLock   sync.RWMutex
	fingerTable       [M]string
	fingerTableLock   sync.RWMutex
	successorList     [successorListLen]string
	successorListLock sync.RWMutex
}

func (n *ChordNode) findSuccessor() {
	
}

func (n *ChordNode) findPredecessor() {}

func (n *ChordNode) closestPrecedingFinger() {}

func (n *ChordNode) join(addr string) bool {}

func (n *ChordNode) stabilize() {}

func (n *ChordNode) notify() {}

func (n *ChordNode) fixFingers() {}