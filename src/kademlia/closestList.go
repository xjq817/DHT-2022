package kademlia

import(

)

type ClosestList struct {
	size int
	addr string
	arr [K]string
}

func (n *ClosestList) insert(addr string) bool {
	if addr=="" {
		return false
	}
	for i:=0;i<n.size;i++ {
		if n.arr[i]==addr {
			return false
		}
	}
	nId:=id(n.addr)
	newDis:=dis(id(addr),nId)
	if (n.size<K){
		for i:=0;i<=n.size;i++ {
			if i==n.size || newDis.Cmp(dis(id(n.arr[i]),nId))<0 {
				for j:=n.size;j>i;j-- {
					n.arr[j]=n.arr[j-1]
				}
				n.arr[i]=addr
				n.size++
				return true
			}
		}
	}
	for i:=0;i<K;i++ {
		if newDis.Cmp(dis(id(n.arr[i]),nId))<0 {
			for j:=K-1;j>i;j-- {
				n.arr[j]=n.arr[j-1]
			}
			n.arr[i]=addr
			return true
		}
	}
	return false
}

func (n *ClosestList) remove(addr string) bool {
	for i:=0;i<n.size;i++ {
		if n.arr[i]==addr {
			for j:=i;j<n.size;j++ {
				n.arr[j]=n.arr[j+1]
			}
			n.size--
			return true
		}
	}
	return false
}