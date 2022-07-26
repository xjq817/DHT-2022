package kademlia

import(

)

type Bucket struct {
	size int
	arr [K]string
}

func (n *Bucket) renew() {
	for i:=0;i<n.size;i++ {
		if !Ping(n.arr[i]) {
			for j:=i+1;j<n.size;j++ {
				n.arr[j-1]=n.arr[j]
			}
			n.size--
			return
		}
	}
}

func (n *Bucket) update(addr string) {
	if addr=="" {
		return
	}
	pos:=-1
	for i:=0;i<n.size;i++ {
		if n.arr[i]==addr {
			pos=i;break;
		}
	}
	if pos!=-1 {
		for i:=pos+1;i<n.size;i++ {
			n.arr[i-1]=n.arr[i]
		}
		n.arr[n.size-1]=addr
		return
	}
	if n.size<K {
		n.arr[n.size]=addr
		n.size++;
		return
	}
	if Ping(n.arr[0]) {
		addr=n.arr[0]
	}
	for i:=1;i<K;i++ {
		n.arr[i-1]=n.arr[i]
	}
	n.arr[K-1]=addr
}