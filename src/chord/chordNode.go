package chord

import(
	"errors"
	"math/big"
	"net"
	"net/rpc"
	"sync"
	"time"
)

type ChordNode struct {
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

func (n *ChordNode) run() {
	n.server=rpc.NewServer()
	err:=n.server.Register(n)
	if err!=nil {
		return
	}
	n.listener,err=net.Listen("tcp",n.addr)
	if err!=nil {
		return
	}
	go MyAccept(n.server,n.listener,n)
	n.maintain()
}

func (n *ChordNode) create() {
	n.onlineLock.Lock()
	n.online=true
	n.onlineLock.Unlock()

	n.successorListLock.Lock()
	n.successorList[0]=n.addr
	n.successorListLock.Unlock()

	n.predecessorLock.Lock()
	n.predecessor=n.addr
	n.predecessorLock.Unlock()

	n.fingerTableLock.Lock()
	for i:=0;i<M;i++ {
		n.fingerTable[i]=n.addr
	}
	n.fingerTableLock.Unlock()
}

func (n *ChordNode) GetSuccessorList(_ string,reply *[successorListLen]string) error {
	n.successorListLock.RLock()
	*reply=n.successorList
	n.successorListLock.RUnlock()
	return nil
}

func (n *ChordNode) join(addr string) bool {
	if n.online{
		return false
	}
	n.predecessorLock.Lock()
	n.predecessor=""
	n.predecessorLock.Unlock()
	var suc string
	err := RemoteCall(addr,"ChordNode.FindSuccessor",id(n.addr),&suc)
	if err!=nil {
		return false
	}
	var sucList [successorListLen]string
	err = RemoteCall(suc,"ChordNode.GetSuccessorList","",&sucList)
	if err!=nil {
		return false
	}
	n.successorListLock.Lock()
	n.successorList[0]=suc
	cnt:=1
	for i:=0;i<successorListLen && cnt<successorListLen;i++ {
		if Ping(sucList[i]) {
			n.successorList[cnt]=sucList[i]
			cnt++
		}
	}
	n.successorListLock.Unlock()
	if suc!=n.addr {
		n.storeLock.Lock()
		err=RemoteCall(suc,"ChordNode.TransferData",n.addr,&n.store)
		n.storeLock.Unlock()
		if err!=nil {
			return false
		}
	}
	n.fingerTableLock.Lock()
	n.fingerTable[0]=suc
	n.fingerTableLock.Unlock()
	nId:=id(n.addr)
	for i:=1;i<M;i++ {
		var fingerNodeIth string
		err=RemoteCall(suc,"ChordNode.FindSuccessor",whereMod(nId,i),&fingerNodeIth)
		if err!=nil {
			fingerNodeIth=""
		}
		n.fingerTableLock.Lock()
		n.fingerTable[i]=fingerNodeIth
		n.fingerTableLock.Unlock()
	}
	n.onlineLock.Lock()
	n.online=true
	n.onlineLock.Unlock()
	return true
}

func (n *ChordNode) DeletePreStore(toStore *map[string]string,_ *string) error {
	n.preStoreLock.Lock()
	for i,_:=range *toStore {
		delete(n.preStore,i)
	}
	n.preStoreLock.Unlock()
	return nil
}

func (n *ChordNode) TransferData(to string,toStore *map[string]string) error {
	toId:=id(to)
	nId:=id(n.addr)
	n.storeLock.Lock()
	n.preStoreLock.Lock()
	n.preStore=make(map[string]string)
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
	err:=n.FindItsSuccessor("",&suc)
	if err!=nil {
		return err
	}
	if suc!=to {
		err=RemoteCall(suc,"ChordNode.DeletePreStore",toStore,nil)
	}
	return nil
}

func (n *ChordNode) FindSuccessor(keyId *big.Int,reply *string) error {
	var suc string
	err:=n.FindItsSuccessor("",&suc)
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
	return RemoteCall(pre,"ChordNode.FindSuccessor",keyId,reply)
}

func (n *ChordNode) FindItsSuccessor(_ string,reply *string) error {
	var suc string
	for i:=0;i<successorListLen;i++ {
		n.successorListLock.RLock()
		suc=n.successorList[i]
		n.successorListLock.RUnlock()
		if suc!="" && Ping(suc) {
			*reply=suc
			if i>0 {
				n.successorListLock.Lock()
				for j:=i;j<successorListLen;j++ {
					n.successorList[j-i]=n.successorList[j]
				}
				n.successorListLock.Unlock()
				time.Sleep(maintainTime)
				_ = RemoteCall(suc,"chord.Notify",n.addr,nil)
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
	defer n.fingerTableLock.RUnlock()
	for i:=M-1;i>=0;i-- {
		fin:=n.fingerTable[i]
		if Ping(fin) && isIn(id(fin),nId,keyId,false){
			return fin,nil
		}
	}
	var suc string
	err:=n.FindItsSuccessor("",&suc)
	if err!=nil {
		return "",errors.New("not found")
	}
	return suc,nil
}

func (n *ChordNode) GetPredecessor(_ string,reply *string) error {
	n.predecessorLock.RLock()
	*reply=n.predecessor
	n.predecessorLock.RUnlock()
	return nil
}

func (n *ChordNode) GetStore(_ string,reply *map[string]string) error {
	n.storeLock.RLock()
	for i,v:=range n.store {
		(*reply)[i]=v
	}
	n.storeLock.RUnlock()
	return nil
}

func (n *ChordNode) Stabilize(_ string,_ *string) error {
	var suc string
	err:=n.FindItsSuccessor("",&suc)
	if err!=nil{
		return nil
	}
	var sucPre string
	_=RemoteCall(suc,"ChordNode.GetPredecessor","",&sucPre)
	if Ping(sucPre) && isIn(id(sucPre),id(n.addr),id(suc),false) {
		suc=sucPre
	}
	var sucList [successorListLen]string
	_=RemoteCall(suc,"ChordNode.GetSuccessorList","",&sucList)
	n.successorListLock.Lock()
	n.successorList[0]=suc
	cnt:=1
	for i:=0;i<successorListLen && cnt<successorListLen;i++ {
		if Ping(sucList[i]) {
			n.successorList[cnt]=sucList[i]
			cnt++
		}
	}
	n.successorListLock.Unlock()
	_=RemoteCall(suc,"ChordNode.Notify",n.addr,nil)
	return nil
}

func (n *ChordNode) ping(addr string) bool {
	return Ping(addr)
}

func (n *ChordNode) Notify(mayPre string,_ *string) error {
	n.predecessorLock.RLock()
	pre:=n.predecessor
	n.predecessorLock.RUnlock()
	if pre=="" || pre!=mayPre && isIn(id(mayPre),id(pre),id(n.addr),false) {
		n.predecessorLock.Lock()
		n.predecessor=mayPre
		n.predecessorLock.Unlock()
		n.preStoreLock.Lock()
		n.preStore=make(map[string]string)
		_=RemoteCall(mayPre,"ChordNode.GetStore","",&n.preStore)
		n.preStoreLock.Unlock()
	}
	return nil
}

func (n *ChordNode) delete(key string) bool {
	if !n.online {
		return false
	}
	var suc string
	err:=n.FindSuccessor(id(key),&suc)
	if err!=nil {
		return false
	}
	err=RemoteCall(suc,"ChordNode.DeleteInStore",key,nil)
	if err!=nil {
		return false
	}
	return true
}

func (n *ChordNode) DeleteInStore(key string,_ *string) error {
	n.storeLock.Lock()
	_,isExist:=n.store[key]
	delete(n.store,key)
	n.storeLock.Unlock()
	if !isExist {
		return errors.New("delete no exist in store")
	}
	var suc string
	err:=n.FindItsSuccessor("",&suc)
	if err!=nil {
		return err
	}
	err=RemoteCall(suc,"ChordNode.DeleteInPreStore",key,nil)
	return err
}

func (n *ChordNode) DeleteInPreStore(key string,_ *string) error {
	n.preStoreLock.Lock()
	_,isExist:=n.preStore[key]
	delete(n.preStore,key)
	n.preStoreLock.Unlock()
	if !isExist {
		return errors.New("delete no exist in preStore")
	}
	return nil
}

func (n *ChordNode) fixFingers() {
	needFix:=whereMod(id(n.addr),n.fixNow)
	var suc string
	err:=n.FindSuccessor(needFix,&suc)
	if err!=nil {
		return
	}
	n.fingerTableLock.Lock()
	n.fingerTable[n.fixNow]=suc
	n.fingerTableLock.Unlock()
	n.fixNow=(n.fixNow+1)%M
}

func (n *ChordNode) PutIntoPreStore(storeData *map[string]string,_ *string) error {
	n.preStoreLock.Lock()
	for i,v := range *storeData {
		n.preStore[i]=v
	}
	n.preStoreLock.Unlock()
	return nil
}

func (n *ChordNode) CheckPredecessor (_ string,_ *string) error {
	n.predecessorLock.RLock()
	pre:=n.predecessor
	n.predecessorLock.RUnlock()
	if pre!="" && !Ping(pre) {
		n.predecessorLock.Lock()
		n.predecessor=""
		n.predecessorLock.Unlock()
		n.storeLock.Lock()
		n.preStoreLock.RLock()
		for i,v:=range n.preStore {
			n.store[i]=v
		}
		n.storeLock.Unlock()
		n.preStoreLock.RUnlock()
		var suc string
		err:=n.FindItsSuccessor("",&suc)
		if err!=nil {
			return nil
		}
		if suc!=n.addr {
			n.preStoreLock.Lock()
			_=RemoteCall(suc,"ChordNode.PutIntoPreStore",&n.preStore,nil)
			n.preStore=make(map[string]string)
			n.preStoreLock.Unlock()
		}
	}
	return nil
}

func (n *ChordNode) maintain() {
	go func() {
		for {
			if n.online {
				_=n.Stabilize("",nil)
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
	go func() {
		for {
			if n.online {
				_=n.CheckPredecessor("",nil)
			}
			time.Sleep(maintainTime)
		}
	}()
}

func (n *ChordNode) clear() {
	n.storeLock.Lock()
	n.store=make(map[string]string)
	n.storeLock.Unlock()
	n.preStoreLock.Lock()
	n.preStore=make(map[string]string)
	n.preStoreLock.Unlock()
	n.quitSignal=make(chan bool,2)
}

func (n *ChordNode) quit() {
	if !n.online {
		return
	}
	n.shutDownServer()
	var suc string
	err:=n.FindItsSuccessor("",&suc)
	if err!=nil {
		return
	}
	err=RemoteCall(suc,"ChordNode.CheckPredecessor","",nil)
	if err!=nil {
		return
	}
	n.predecessorLock.RLock()
	pre:=n.predecessor
	n.predecessorLock.RUnlock()
	err=RemoteCall(pre,"ChordNode.Stabilize","",nil)
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
	_=n.listener.Close()
}

func (n *ChordNode) forceQuit() {
	if !n.online {
		return
	}
	n.shutDownServer()
	n.clear()
}

func (n *ChordNode) StoreData(dataPair Pair,_ *string) error {
	n.storeLock.Lock()
	n.store[dataPair.Key]=dataPair.Value
	n.storeLock.Unlock()
	var suc string
	err:=n.FindItsSuccessor("",&suc)
	if err!=nil {
		return err
	}
	_=RemoteCall(suc,"ChordNode.PreStoreData",dataPair,nil)
	return nil
}

func (n *ChordNode) PreStoreData(dataPair Pair,_ *string) error {
	n.preStoreLock.Lock()
	n.preStore[dataPair.Key]=dataPair.Value
	n.preStoreLock.Unlock()
	return nil
}

func (n *ChordNode) put(key string,value string) bool {
	if !n.online {
		return false
	}
	var suc string
	err:=n.FindSuccessor(id(key),&suc)
	if err!=nil {
		return false
	}
	err=RemoteCall(suc,"ChordNode.StoreData",Pair{Key:key,Value:value},nil)
	if err!=nil {
		return false
	}
	return true
}

func (n *ChordNode) get(key string) (bool,string) {
	if !n.online {
		return false,""
	}
	var suc string
	err:=n.FindSuccessor(id(key),&suc)
	if err!=nil {
		return false,""
	}
	var value string
	err=RemoteCall(suc,"ChordNode.FindValue",key,&value)
	if err!=nil {
		return false,""
	}
	return true,value
}

func (n *ChordNode) FindValue(key string,value *string) error {
	var isExist bool
	n.storeLock.RLock()
	*value,isExist=n.store[key]
	n.storeLock.RUnlock()
	if !isExist {
		*value=""
		return errors.New("no found")
	}
	return nil
}