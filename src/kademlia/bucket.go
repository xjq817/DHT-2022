package kademlia

type Bucket struct {
	Size int
	Arr  [K]string
}

func (n *Bucket) renew() {
	for i := 0; i < n.Size; i++ {
		if !Ping(n.Arr[i]) {
			for j := i + 1; j < n.Size; j++ {
				n.Arr[j-1] = n.Arr[j]
			}
			n.Size--
			return
		}
	}
}

func (n *Bucket) update(addr string) {
	if addr == "" {
		return
	}
	pos := -1
	for i := 0; i < n.Size; i++ {
		if n.Arr[i] == addr {
			pos = i
			break
		}
	}
	if pos != -1 {
		for i := pos + 1; i < n.Size; i++ {
			n.Arr[i-1] = n.Arr[i]
		}
		n.Arr[n.Size-1] = addr
		return
	}
	if n.Size < K {
		n.Arr[n.Size] = addr
		n.Size++
		return
	}
	if Ping(n.Arr[0]) {
		addr = n.Arr[0]
	}
	for i := 1; i < K; i++ {
		n.Arr[i-1] = n.Arr[i]
	}
	n.Arr[K-1] = addr
}
