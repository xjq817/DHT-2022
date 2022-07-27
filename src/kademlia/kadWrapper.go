package kademlia

type KadWrapper struct {
	node *KadNode
}

func (w *KadWrapper) Initialize(addr string) {
	w.node = new(KadNode)
	w.node.initialize(addr)
}

func (w *KadWrapper) Run() {
	w.node.run()
}

func (w *KadWrapper) Create() {
	w.node.create()
}

func (w *KadWrapper) Join(addr string) bool {
	return w.node.join(addr)
}

func (w *KadWrapper) Quit() {
	w.node.quit()
}

func (w *KadWrapper) ForceQuit() {
	w.node.forceQuit()
}

func (w *KadWrapper) Ping(addr string) bool {
	return w.node.ping(addr)
}

func (w *KadWrapper) Put(key string, value string) bool {
	return w.node.put(key, value)
}

func (w *KadWrapper) Get(key string) (bool, string) {
	return w.node.get(key)
}

func (w *KadWrapper) Delete(key string) bool {
	return w.node.delete(key)
}
