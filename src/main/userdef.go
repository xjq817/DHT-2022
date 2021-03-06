package main

/* In this file, you should implement function "NewNode" and
 * a struct which implements the interface "dhtNode".
 */

import (
	"kademlia"
	"strconv"
)

func NewNode(port int) dhtNode {
	// Todo: create a node and then return it.
	var w kademlia.KadWrapper
	w.Initialize(GetLocalAddress() + ":" + strconv.Itoa(port))
	return &w
}

// Todo: implement a struct which implements the interface "dhtNode".
