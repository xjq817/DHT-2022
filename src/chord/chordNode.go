package chord

import(
	"errors"
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

	server            *rpc.Server
	listener          net.Listener

	quitSignal        chan bool

	fixNow            int
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

func (n *ChordNode) run() {
	n.initializeServer()
	n.maintain()
}

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

func (n *ChordNode) getSuccessorList (_ string,reply *[successorListLen]string) error {
	n.successorListLock.RLock()
	*reply=n.successorList
	n.successorListLock.RUnlock()
	return nil
}

func (n *ChordNode) join(addr string) bool {
	n.onlineLock.RLock()
	if n.online{
		n.onlineLock.RUnlock()
		return false
	}
	n.onlineLock.RUnlock()
	n.predecessorLock.Lock()
	n.predecessor=""
	n.predecessorLock.Unlock()
	var suc string
	err := remoteCall(n.addr,"chordNode.findSuccessor",id(n.addr),&suc)
	if err!=nil {
		return false
	}
	var sucList [successorListLen]string
	err = remoteCall(suc,"chordNode.getSuccessorList","",&sucList)
	n.successorListLock.Lock()
	if err!=nil {
		return false
	}
	n.successorList[0]=suc
	cnt:=1
	for i:=1;i<successorListLen;i++ {
		if Ping(sucList[i-1]) {
			n.successorList[cnt]=sucList[i-1]
			cnt++
		}
	}
	n.successorListLock.Unlock()
	if suc!=n.addr {
		n.storeLock.Lock()
		err=remoteCall(suc,"chordNode.transferData",n.addr,&n.store)
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
		if err!=nil {
			fingerNodeIth=""
		}
		n.fingerTableLock.Lock()
		n.fingerTable[i]=fingerNodeIth
		n.fingerTableLock.Unlock()
	}
	return true
}

func (n *ChordNode) deletePreStore(toStore *map[string]string,_ *string) error {
	n.preStoreLock.Lock()
	for i,_:=range *toStore {
		delete(n.preStore,i)
	}
	n.preStoreLock.Unlock()
	return nil
}

func (n *ChordNode) transferData(to string,toStore *map[string]string) error {
	toId:=id(to)
	nId:=id(n.addr)
	n.storeLock.Lock()
	n.preStoreLock.Lock()
	for i,v:=range n.store {
		if !isIn(id(i),toId,nId,true) {
			(*toStore)[i]=v
			n.preStore[i]=v
			delete(n.store,i)
		}
	}
	n.storeLock.Unlock()
	n.preStoreLock.Unlock()
	var suc string
	err:=n.findItsSuccessor("",&suc)
	if err!=nil {
		return err
	}
	if suc!=to {
		err=remoteCall(suc,"chordNode.deletePreStore",toStore,nil)
	}
	return nil
}

func (n *ChordNode) findSuccessor(keyId *big.Int,reply *string) error {
	var suc string
	err:=n.findItsSuccessor("",&suc)
	if err!=nil {
		return err
	}
	if (isIn(keyId,id(n.addr),id(suc),true)){
		*reply=suc
		return nil
	}
	var pre string
	pre,err=n.closestPrecedingFinger(keyId)
	if err!=nil {
		return err
	}
	return remoteCall(pre,"chordNode.findSuccessor",keyId,reply)
}

func (n *ChordNode) findItsSuccessor(request string,reply *string) error {
	var suc string
	for i:=0;i<successorListLen;i++ {
		n.successorListLock.RLock()
		suc=n.successorList[i]
		n.successorListLock.RUnlock()
		if Ping(suc) {
			*reply=suc
			if i>0 {
				n.successorListLock.Lock()
				for j:=i;j<successorListLen;j++ {
					n.successorList[j-i]=n.successorList[j]
				}
				n.successorListLock.Unlock()
				time.Sleep(maintainTime)
				_ = remoteCall(suc,"chord.Notify",n.addr,nil)
			}
			return nil
		}
	}
	*reply=""
	return errors.New("no successor")
}

func (n *ChordNode) closestPrecedingFinger(keyId *big.Int) (string,error) {
	nId:=id(n.addr)
	n.fingerTableLock.RLock()
	for i:=M-1;i>=0;i-- {
		fin:=n.fingerTable[i]
		if Ping(fin) && isIn(id(fin),nId,keyId,false){
			n.fingerTableLock.RUnlock()
			return fin,nil
		}
	}
	n.fingerTableLock.RUnlock()
	var suc string
	err:=n.findItsSuccessor("",&suc)
	if err!=nil {
		return "",errors.New("not found")
	}
	return suc,nil
}

func (n *ChordNode) getPredecessor(_ string,reply *string) error {
	n.predecessorLock.RLock()
	*reply=n.predecessor
	n.predecessorLock.RUnlock()
	return nil
}

func (n *ChordNode) getStore(_ string,reply *map[string]string) error {
	n.storeLock.RLock()
	*reply=make(map[string]string)
	for i,v:=range n.store {
		(*reply)[i]=v
	}
	n.storeLock.RUnlock()
	return nil
}

func (n *ChordNode) stabilize(_ string,_ *string) error {
	var suc string
	err:=n.findItsSuccessor("",&suc)
	if err!=nil{
		return err
	}
	var sucPre string
	err=remoteCall(suc,"chordNode.getPredecessor","",&sucPre)
	if err!=nil {
		return err
	}
	if Ping(sucPre) && isIn(id(sucPre),id(n.addr),id(suc),false) {
		suc=sucPre
	}
	var sucList [successorListLen]string
	err=remoteCall(suc,"chordNode.getSuccessorList","",&sucList)
	if err!=nil {
		return err
	}
	n.successorListLock.Lock()
	n.successorList[0]=suc
	cnt:=1
	for i:=1;i<successorListLen;i++ {
		if Ping(sucList[i-1]) {
			n.successorList[cnt]=sucList[i-1]
			cnt++
		}
	}
	n.successorListLock.Unlock()
	err=remoteCall(suc,"chordNode.notify",n.addr,nil)
	if err!=nil {
		return err
	}
	return nil
}

func (n *ChordNode) ping(addr string) bool {
	return Ping(addr)
}

func (n *ChordNode) notify(mayPre string,reply *string) error {
	n.predecessorLock.RLock()
	pre:=n.predecessor
	n.predecessorLock.RUnlock()
	if pre==mayPre {
		return nil
	}
	if pre=="" || isIn(id(mayPre),id(pre),id(n.addr),false) {
		n.predecessorLock.Lock()
		n.predecessor=mayPre
		n.predecessorLock.Unlock()
		n.preStoreLock.Lock()
		err:=remoteCall(mayPre,"chordNode.getStore","",&n.preStore)
		n.preStoreLock.Unlock()
		if err!=nil {
			return err
		}
	}
	return nil
}

func (n *ChordNode) delete(key string) bool {
	if !n.online {
		return false
	}
	var suc string
	err:=n.findSuccessor(id(key),&suc)
	if err!=nil {
		return false
	}
	err=remoteCall(suc,"chordNode.deleteInStore",key,nil)
	if err!=nil {
		return false
	}
	return true
}

func (n *ChordNode) deleteInStore(key string,_ *string) error {
	n.storeLock.Lock()
	_,isExist:=n.store[key]
	delete(n.store,key)
	n.storeLock.Unlock()
	if !isExist {
		return errors.New("delete key in store no exist")
	}
	var suc string
	err:=n.findItsSuccessor("",&suc)
	if err!=nil {
		return err
	}
	err=remoteCall(suc,"chordNode.deleteInPreStore",key,nil)
	if err!=nil {
		return err
	}
	return nil
}

func (n *ChordNode) deleteInPreStore(key string,_ *string) error {
	n.preStoreLock.Lock()
	_,isExist:=n.preStore[key]
	delete(n.preStore,key)
	n.preStoreLock.Unlock()
	if !isExist {
		return errors.New("delete key in preStore no exist")
	}
	return nil
}

func (n *ChordNode) fixFingers() {
	needFix:=whereMod(id(n.addr),n.fixNow)
	var suc string
	err:=n.findSuccessor(needFix,&suc)
	if err!=nil {
		return
	}
	n.fingerTableLock.Lock()
	n.fingerTable[n.fixNow]=suc
	n.fingerTableLock.Unlock()
	n.fixNow=(n.fixNow+1)%M
}

// func (n *ChordNode) checkPredecessor (_ string,_ *string) error {
// 	n.predecessorLock.RLock()
// 	pre:=n.predecessor
// 	n.predecessorLock.RUnlock()
// 	if pre!="" && !Ping(pre) {
// 		n.predecessorLock.Lock()
// 		n.predecessor=""
// 		n.predecessorLock.Unlock()
// 		///////////////////////
// 	}
// }

func (n *ChordNode) maintain() {
	go func() {
		var emp string
		for {
			if n.online {
				n.stabilize("",&emp)
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
	// go func() {
	// 	var emp string
	// 	for {
	// 		if n.online {
	// 			_:=n.checkPredecessor("",&emp)
	// 		}
	// 		time.Sleep(maintainTime)
	// 	}
	// }()
}

func (n *ChordNode) clear() {
	n.storeLock.Lock()
	n.store=make(map[string]string)
	n.storeLock.Unlock()
	n.preStoreLock.Lock()
	n.preStore=make(map[string]string)
	n.predecessorLock.Unlock()
	n.quitSignal=make(chan bool,2)
}

func (n *ChordNode) quit() {
	n.onlineLock.RLock()
	if !n.online {
		n.onlineLock.RUnlock()
		return
	}
	n.onlineLock.RUnlock()
	n.shutDownServer()
	n.predecessorLock.RLock()
	pre:=n.predecessor
	n.predecessorLock.RUnlock()
	var suc,emp string
	err:=n.findItsSuccessor("",&suc)
	if err!=nil {
		return
	}
	err=remoteCall(pre,"chordNode.stabilize","",&emp)
	if err!=nil {
		return
	}
	n.clear()
}

func (n *ChordNode) shutDownServer() {
	n.onlineLock.Lock()
	n.online=false
	n.onlineLock.Unlock()
	n.quitSignal <-true
	err:=n.listener.Close()
	if err!=nil {
		return
	}
}

func (n *ChordNode) forceQuit() {
	n.onlineLock.RLock()
	if !n.online {
		n.onlineLock.RUnlock()
		return
	}
	n.onlineLock.RUnlock()
	n.shutDownServer()
	n.clear()
}

func (n *ChordNode) storeData(dataPair Pair,_ *string) error {
	n.storeLock.Lock()
	n.store[dataPair.Key]=dataPair.Value
	n.storeLock.Unlock()
	var suc string
	err:=n.findItsSuccessor("",&suc)
	if err!=nil {
		return err
	}
	err=remoteCall(suc,"chordNode.preStoreData",dataPair,nil)
	if err!=nil {
		return err
	}
	return nil
}

func (n *ChordNode) preStoreData(dataPair Pair,_ *string) error {
	n.preStoreLock.Lock()
	n.preStore[dataPair.Key]=dataPair.Value
	n.preStoreLock.Unlock()
	return nil
}

func (n *ChordNode) put(key string,value string) bool {
	n.onlineLock.RLock()
	if !n.online {
		n.onlineLock.RUnlock()
		return false
	}
	n.onlineLock.RUnlock()
	var suc string
	err:=n.findSuccessor(id(key),&suc)
	if err!=nil {
		return false
	}
	var dataPair=Pair{key,value}
	err=remoteCall(suc,"chordNode.storeData",dataPair,nil)
	if err!=nil {
		return false
	}
	return true
}

func (n *ChordNode) get(key string) (bool,string) {
	n.onlineLock.RLock()
	if !n.online {
		n.onlineLock.RUnlock()
		return false,""
	}
	n.onlineLock.RUnlock()
	var suc string
	err:=n.findSuccessor(id(key),&suc)
	if err!=nil {
		return false,""
	}
	var value string
	err=remoteCall(suc,"chordNode.findValue",key,&value)
	if err!=nil {
		return false,""
	}
	return true,value
}

func (n *ChordNode) findValue(key string,value *string) error {
	var isExist bool
	n.storeLock.RLock()
	*value,isExist=n.store[key]
	n.storeLock.RUnlock()
	if !isExist {
		return errors.New("no found")
	}
	return nil
}