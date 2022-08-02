package kademlia

import (
	"errors"
	"net"
	"net/rpc"
	"sync"
	"time"
)

type KadNode struct {
	addr          string
	online        bool
	onlineLock    sync.RWMutex
	store         map[string]string
	republishTime map[string]time.Time
	expireTime    map[string]time.Time
	dataLock      sync.RWMutex
	kbucket       [M]Bucket
	kbucketLock   sync.RWMutex
	server        *rpc.Server
	listener      net.Listener
	quitSignal    chan bool
}

func (n *KadNode) clear() {
	n.dataLock.Lock()
	n.store = make(map[string]string)
	n.republishTime = make(map[string]time.Time)
	n.expireTime = make(map[string]time.Time)
	n.dataLock.Unlock()
	n.quitSignal = make(chan bool, 2)
}

func (n *KadNode) copy() map[string]string {
	n.dataLock.RLock()
	tmp := make(map[string]string)
	for key, value := range n.store {
		tmp[key] = value
	}
	n.dataLock.RUnlock()
	return tmp
}

func (n *KadNode) republish() []string {
	var list []string
	n.dataLock.RLock()
	for key, tim := range n.republishTime {
		if time.Now().After(tim) {
			list = append(list, key)
		}
	}
	n.dataLock.RUnlock()
	return list
}

func (n *KadNode) expire() {
	var list []string
	n.dataLock.RLock()
	for key, tim := range n.expireTime {
		if time.Now().After(tim) {
			list = append(list, key)
		}
	}
	n.dataLock.RUnlock()
	n.dataLock.Lock()
	for _, key := range list {
		delete(n.store, key)
		delete(n.republishTime, key)
		delete(n.expireTime, key)
	}
	n.dataLock.Unlock()
}

func (n *KadNode) maintain() {
	for {
		if n.online {
			n.kbucketLock.Lock()
			for i := 0; i < M; i++ {
				n.kbucket[i].renew()
			}
			n.kbucketLock.Unlock()
			storeDate := n.copy()
			republishList := n.republish()
			for _, key := range republishList {
				n.put(key, storeDate[key])
			}
			n.expire()
		}
		time.Sleep(maintainTime)
	}
}

func (n *KadNode) initialize(addr string) {
	n.addr = addr
	n.clear()
}

func (n *KadNode) run() {
	n.server = rpc.NewServer()
	err := n.server.Register(n)
	if err != nil {
		return
	}
	n.listener, err = net.Listen("tcp", n.addr)
	if err != nil {
		return
	}
	go MyAccept(n.server, n.listener, n)
	go n.maintain()
}

func (n *KadNode) create() {
	n.onlineLock.Lock()
	n.online = true
	n.onlineLock.Unlock()
}

func (n *KadNode) kbucketUpdate(addr string) {
	if addr == "" || addr == n.addr {
		return
	}
	n.kbucketLock.Lock()
	n.kbucket[cpl(id(n.addr), id(addr))].update(addr)
	n.kbucketLock.Unlock()
}

func (n *KadNode) join(addr string) bool {
	if n.online {
		return false
	}
	n.kbucketUpdate(addr)
	var list ClosestList
	err := RemoteCall(addr, "KadNode.FindNode", DataPair{Key: n.addr, Value: n.addr}, &list)
	if err != nil {
		logErrorFunctionCall(n.addr, "KadNode.join", "KadNode.FindNode", err)
		return false
	}
	for i := 0; i < list.Size; i++ {
		n.kbucketUpdate(list.Arr[i])
	}
	list = n.lookUp(n.addr)
	for i := 0; i < list.Size; i++ {
		n.kbucketUpdate(list.Arr[i])
		var tmp ClosestList
		err = RemoteCall(list.Arr[i], "KadNode.FindNode", DataPair{Key: n.addr, Value: n.addr}, &tmp)
		if err != nil {
			logErrorFunctionCall(n.addr, "KadNode.join", "KadNode.FindNode", err)
			continue
		}
		for j := 0; j < tmp.Size; j++ {
			n.kbucketUpdate(tmp.Arr[j])
		}
	}
	n.onlineLock.Lock()
	n.online = true
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
	n.online = false
	n.onlineLock.Unlock()
	n.quitSignal <- true
	_ = n.listener.Close()
}

func (n *KadNode) forceQuit() {
	n.quit()
}

func (n *KadNode) ping(addr string) bool {
	return Ping(addr)
}

func (n *KadNode) put(key string, value string) bool {
	if !n.online {
		return false
	}
	list := n.lookUp(key)
	list.insert(n.addr)
	for i := 0; i < list.Size; i++ {
		err := RemoteCall(list.Arr[i], "KadNode.Store", StorePair{Value: DataPair{Key: key, Value: value}, Addr: n.addr}, nil)
		if err != nil {
			logErrorFunctionCall(n.addr, "kadNode.put", "KadNode.Store", err)
		}
	}
	return true
}

func (n *KadNode) get(key string) (bool, string) {
	if !n.online {
		return false, ""
	}
	var getPair FindValuePair
	n.FindValue(DataPair{Key: key, Value: n.addr}, &getPair)
	if getPair.Value != "" {
		return true, getPair.Value
	}
	isFind := make(map[string]bool)
	isFind[n.addr] = true
	list := getPair.List
	list.insert(n.addr)
	flag := true
	for flag {
		flag = false
		var tmp ClosestList
		tmp.Addr = key
		var delList []string
		for i := 0; i < list.Size; i++ {
			if isFind[list.Arr[i]] {
				continue
			}
			isFind[list.Arr[i]] = true
			var getPairI FindValuePair
			err := RemoteCall(list.Arr[i], "KadNode.FindValue", DataPair{Key: key, Value: n.addr}, &getPairI)
			if err != nil {
				logErrorFunctionCall(n.addr, "KadNode.get", "KadNode.FindValue", err)
				delList = append(delList, list.Arr[i])
				continue
			}
			if getPairI.Value != "" {
				return true, getPairI.Value
			}
			for j := 0; j < getPairI.List.Size; j++ {
				tmp.insert(getPairI.List.Arr[j])
			}
		}
		for _, value := range delList {
			list.remove(value)
		}
		for i := 0; i < tmp.Size; i++ {
			if list.insert(tmp.Arr[i]) {
				flag = true
			}
		}
	}
	list = n.lookUp(key)
	for i := 0; i < list.Size; i++ {
		var getPairI FindValuePair
		_ = RemoteCall(list.Arr[i], "Kad.FindValue", DataPair{Key: key, Value: n.addr}, &getPairI)
		if getPairI.Value != "" {
			return true, getPairI.Value
		}
	}
	return false, ""
}

func (n *KadNode) delete(key string) bool {
	return true
}

func (n *KadNode) Store(storePair StorePair, _ *string) error {
	n.dataLock.Lock()
	n.store[storePair.Value.Key] = storePair.Value.Value
	n.republishTime[storePair.Value.Key] = time.Now().Add(RepublishTime)
	n.expireTime[storePair.Value.Key] = time.Now().Add(ExpireTime)
	n.dataLock.Unlock()
	n.kbucketUpdate(storePair.Addr)
	return nil
}

func (n *KadNode) FindNode(dataPair DataPair, closestList *ClosestList) error {
	closestList.Addr = dataPair.Key
	if !n.online {
		return errors.New("findnode offline")
	}
	n.kbucketLock.RLock()
	len := cpl(id(n.addr), id(dataPair.Key))

	for i := 0; i < M; i++ {
		if len+i < M {
			for j := 0; j < n.kbucket[len+i].Size; j++ {
				if Ping(n.kbucket[len+i].Arr[j]) {
					closestList.insert(n.kbucket[len+i].Arr[j])
				}
			}
		}
		if len-i >= 0 && i > 0 {
			for j := 0; j < n.kbucket[len-i].Size; j++ {
				if Ping(n.kbucket[len-i].Arr[j]) {
					closestList.insert(n.kbucket[len-i].Arr[j])
				}
			}
		}
		if closestList.Size == K {
			break
		}
	}

	// for i := 0; i < M; i++ {
	// 	for j := 0; j < n.kbucket[i].Size; j++ {
	// 		if Ping(n.kbucket[i].Arr[j]) {
	// 			closestList.insert(n.kbucket[i].Arr[j])
	// 		}
	// 	}
	// }
	n.kbucketLock.RUnlock()
	if dataPair.Value != n.addr {
		n.kbucketUpdate(dataPair.Value)
	}
	return nil
}

func (n *KadNode) FindValue(dataPair DataPair, reply *FindValuePair) error {
	n.dataLock.RLock()
	defer n.dataLock.RUnlock()
	reply.List.Addr = dataPair.Key
	if !n.online {
		return errors.New("FindValue no online")
	}
	value, isExist := n.store[dataPair.Key]

	if isExist {
		reply.Value = value
		if dataPair.Value != n.addr {
			n.kbucketUpdate(dataPair.Value)
		}
		return nil
	}
	n.kbucketLock.RLock()
	for i := 0; i < M; i++ {
		for j := 0; j < n.kbucket[i].Size; j++ {
			if Ping(n.kbucket[i].Arr[j]) {
				reply.List.insert(n.kbucket[i].Arr[j])
			}
		}
	}
	n.kbucketLock.RUnlock()
	if dataPair.Value != n.addr {
		n.kbucketUpdate(dataPair.Value)
	}
	return nil
}

func (n *KadNode) lookUp(addr string) (list ClosestList) {
	if !n.online {
		return
	}
	_ = n.FindNode(DataPair{Key: addr, Value: n.addr}, &list)
	list.insert(n.addr)
	flag := true
	isFind := make(map[string]bool)
	isFind[n.addr] = true
	for flag {
		flag = false
		var tmp ClosestList
		tmp.Addr = addr
		var delList []string
		for i := 0; i < list.Size; i++ {
			if isFind[list.Arr[i]] {
				continue
			}
			n.kbucketUpdate(list.Arr[i])
			isFind[list.Arr[i]] = true
			var listI ClosestList
			err := RemoteCall(list.Arr[i], "KadNode.FindNode", DataPair{Key: addr, Value: n.addr}, &listI)
			if err != nil {
				logErrorFunctionCall(n.addr, "KadNode.lookUp", "KadNode.FindNode", err)
				delList = append(delList, list.Arr[i])
				continue
			}
			for j := 0; j < listI.Size; j++ {
				tmp.insert(listI.Arr[j])
			}
		}
		for _, value := range delList {
			list.remove(value)
		}
		for i := 0; i < tmp.Size; i++ {
			if list.insert(tmp.Arr[i]) {
				flag = true
			}
		}
	}
	return
}
