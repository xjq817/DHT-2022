package kademlia

import(
	"errors"
	"math/big"
	"net"
	"net/rpc"
	"sync"
	"time"
)

type KadNode struct {
	addr string
	online bool
	onlineLock sync.RWMutex
	store map[string]string
	storeLock sync.RWMutex
	kbucket [M]Bucket
	kbucketLock sync.RWMutex
	server *rpc.Server
	listener net.Listener
	quitSignal chan bool
}

func (n *KadNode) clear() {
	n.storeLock.Lock()
	n.store=make(map[string]string)
	n.storeLock.Unlock()
	n.quitSignal=make(chan bool,2)
}

func (n *KadNode) maintain() {
	for {
		if n.online {
			
		}
		time.Sleep(maintainTime)
	}
}

func (n *KadNode) initialize(addr string) {
	n.addr=addr
	n.clear()
}

func (n *KadNode) run() {
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
	go n.maintain()
}

func (n *KadNode) create() {
	n.onlineLock.Lock()
	n.online=true
	n.onlineLock.Unlock()
}

func (n *KadNode) kbucketUpdate(addr string) {
	n.kbucketLock.Lock()
	kbucket[cpl(id(n.addr),addr)].update(addr)
	n.kbucketLock.Unlock()
}

func (n *KadNode) join(addr string) bool {
	if n.online {
		return false
	}
	n.kbucketUpdate(addr)
	var list ClosestList
	err:=RemoteCall(addr,"Kad.FindNode",n.addr,&list)
	if err!=nil {
		return false
	}
	for i:=0;i<list.size;i++ {
		n.kbucketUpdate(list[i])
	}
	list=n.lookUp(n.addr)
	for i:=0;i<list.size;i++ {
		n.kbucketUpdate(list[i])
		var tmp ClosestList
		err=RemoteCall(list[i],"Kad.FindNode",n.addr,&tmp)
		if err!=nil {
			continue
		}
		for j:=0;j<tmp.size;j++ {
			n.kbucketUpdate(tmp[j])
		}
	}
	n.onlineLock.Lock()
	n.online=true
	n.onlineLock.Unlock()
	return true
}

func (n *KadNode) quit() {
	if !n.online {
		return
	}
	n.shutDownServer()
	n.clear()
}

func (n *KadNode) shutDownServer() {
	n.onlineLock.Lock()
	n.online=false
	n.onlineLock.Unlock()
	n.quitSignal <-true
	_=n.listener.Close()
}

func (n *KadNode) forceQuit() {
	n.quit()
}

func (n *KadNode) ping(addr string) bool {
	return Ping(addr)
}

func (n *KadNode) put(key string,value string) bool {
	if !n.online {
		return false
	}
	list:=n.lookUp(key)
	list.insert(n.addr)
	for i:=0;i<list.size;i++ {
		_=RemoteCall(list.arr[i],"KadNode.Store",dataPair{Key:key,Value:value},nil)
	}
	return true
}

func (n *KadNode) get(key string) (bool,string) {
	if !n.online {
		return false,""
	}
	var getPair FindValuePair
	n.FindValue(key,&getPair)
	if getPair.Value!="" {
		return true,getPair.Value
	}
	isFind:=make(map[string]bool)
	isFind[n.addr]=true
	list:=getPair.List
	for flag {
		flag=false
		var tmp ClosestList
		tmp.addr=addr
		var delList []string
		for i:=0;i<list.size;i++ {
			if isFind[list.arr[i]]==true {
				continue
			}
			isFind[list.arr[i]]=true
			var getPairI ClosestList
			err:=RemoteCall(list.arr[i],"KadNode.FindValue",key,&getPairI)
			if err!=nil {
				delList=append(delList,list.arr[i])
				continue
			}
			if getPairI.Value!="" {
				return true,getPairI.Value
			}
			for j:=0;j<listI.size;j++ {
				tmp.insert(listI.arr[i])
			}
		}
		for _,value:=range delList {
			list.remove(value)
		}
		for i:=0;i<tmp.size;i++{
			if list.insert(tmp.arr[i]){
				flag=true
			}
		}
	}
	return false,""
}

func (n *KadNode) delete(key string) bool {}

func (n *KadNode) Store(dataPair DataPair,_ *string) error {
	n.storeLock.Lock()
	n.store[dataPair.Key]=dataPair.Value
	n.storeLock.Unlock()
}

func (n *KadNode) FindNode(addr string,list *ClosestList) error {
	list.addr=addr
	if !n.online {
		return errors.New("FindNode no online")
	}
	n.kbucketLock.RLock()
	for i:=0;i<M;i++ {
		for j:=0;j<n.kbucket[i].size;j++ {
			if Ping(n.kbucket[i].arr[j]) {
				list.insert(n.kbucket[i].arr[j])
			}
		}
	}
	n.kbucketLock.RUnlock()
	return nil
}

func (n *KadNode) FindValue(addr string,reply *FindValuePair) error {
	if !n.online {
		return errors.New("FindValue no online")
	}
	n.storeLock.RLock()
	defer n.storeLock.RUnlock()
	value,isExist:=n.store[addr]
	if isExist {
		(*reply).Value=value
		return nil
	}
	(*reply).List.addr=addr
	n.kbucketLock.RLock()
	for i:=0;i<M;i++{
		for j:=0;j<n.kbucket[i].size;j++{
			if Ping(n.kbucket[i].arr[j]) {
				(*reply).List.insert(n.kbucket[i].arr[j])
			}
		}
	}
	n.kbucketLock.RUnlock()
	return nil
}

func (n* KadNode) lookUp(addr string) ClosestList {
	var list ClosestList
	if !n.online {
		return list
	}
	_=n.FindNode(addr,&list)
	flag:=true
	isFind:=make(map[string]bool)
	isFind[n.addr]=true
	for flag {
		flag=false
		var tmp ClosestList
		tmp.addr=addr
		var delList []string
		for i:=0;i<list.size;i++ {
			if isFind[list.arr[i]]==true {
				continue
			}
			isFind[list.arr[i]]=true
			var listI ClosestList
			err:=RemoteCall(list.arr[i],"KadNode.FindNode",addr,&listI)
			if err!=nil {
				delList=append(delList,list.arr[i])
				continue
			}
			for j:=0;j<listI.size;j++ {
				tmp.insert(listI.arr[i])
			}
		}
		for _,value:=range delList {
			list.remove(value)
		}
		for i:=0;i<tmp.size;i++{
			if list.insert(tmp.arr[i]){
				flag=true
			}
		}
	}
	return list
}