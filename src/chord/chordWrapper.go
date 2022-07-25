package chord

type ChordWrapper struct {
	node *ChordNode
}

func (w *ChordWrapper) Initialize(addr string) {
	w.node = new(ChordNode)
	w.node.initialize(addr)
}

func (w *ChordWrapper) Run() {
	w.node.run()
}

func (w *ChordWrapper) Create() {
	w.node.create()
}

func (w *ChordWrapper) Join(addr string) bool {
	return w.node.join(addr)
}

func (w *ChordWrapper) Quit() {
	w.node.quit()
}

func (w *ChordWrapper) ForceQuit() {
	w.node.forceQuit()
}

func (w *ChordWrapper) Ping(addr string) bool {
	return w.node.ping(addr)
}

func (w *ChordWrapper) Put(key string, value string) bool {
	return w.node.put(key, value)
}

func (w *ChordWrapper) Get(key string) (bool, string) {
	return w.node.get(key)
}

func (w *ChordWrapper) Delete(key string) bool {
	return w.node.delete(key)
}