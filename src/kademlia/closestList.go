package kademlia

type ClosestList struct {
	Size int
	Addr string
	Arr  [K]string
}

func (n *ClosestList) insert(addr string) bool {
	if addr == "" {
		return false
	}
	for i := 0; i < n.Size; i++ {
		if n.Arr[i] == addr {
			return false
		}
	}
	nId := id(n.Addr)
	newDis := dis(id(addr), nId)
	if n.Size < K {
		for i := 0; i <= n.Size; i++ {
			if i == n.Size || newDis.Cmp(dis(id(n.Arr[i]), nId)) < 0 {
				for j := n.Size; j > i; j-- {
					n.Arr[j] = n.Arr[j-1]
				}
				n.Arr[i] = addr
				n.Size++
				return true
			}
		}
	}
	for i := 0; i < K; i++ {
		if newDis.Cmp(dis(id(n.Arr[i]), nId)) < 0 {
			for j := K - 1; j > i; j-- {
				n.Arr[j] = n.Arr[j-1]
			}
			n.Arr[i] = addr
			return true
		}
	}
	return false
}

func (n *ClosestList) remove(addr string) bool {
	for i := 0; i < n.Size; i++ {
		if n.Arr[i] == addr {
			for j := i; j < n.Size; j++ {
				n.Arr[j] = n.Arr[j+1]
			}
			n.Size--
			return true
		}
	}
	return false
}
