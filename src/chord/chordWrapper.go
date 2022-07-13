package chord

import(

)

type ChordWrapper struct{
	node *ChordNode
}

func (w *ChordWrapper) Run() {}

func (w *ChordWrapper) Create() {}

func (w *ChordWrapper) Join(addr string) bool {
	return w.ChordNode.join(addr)
}

func (w *ChordWrapper) Quit() {}

func (w *ChordWrapper) ForceQuit() {}

func (w *ChordWrapper) Ping(addr string) bool {}

func (w *ChordWrapper) Put(key string, value string) bool {}

func (w *ChordWrapper) Get(key string) (bool, string) {}

func (w *ChordWrapper) Delete(key string) bool {}