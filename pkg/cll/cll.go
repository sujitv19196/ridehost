package cll

import (
	"fmt"

	"github.com/sujitv19196/ridehost/internal/types"
)

type Node struct {
	prev     *Node
	next     *Node
	key      string
	nodeinfo types.Node
}

// Initialize a Node with default values
func (curr_node *Node) SetDefaults() {
	curr_node.prev = nil
	curr_node.next = nil
}

// Thread-safe circular doubly linked list
type UniqueCLL struct {
	head    *Node
	tail    *Node
	size    int
	pos_map map[string]*Node
	// mu      sync.Mutex
}

// Initialize a UniqueCLL with default values
func (L *UniqueCLL) SetDefaults() {
	// L.mu = sync.Mutex{}
	// L.mu.Lock()
	// defer L.mu.Unlock()
	L.head = nil
	L.tail = nil
	L.size = 0
	L.pos_map = make(map[string]*Node)
}

func (L *UniqueCLL) Clear() {
	// L.mu.Lock()
	// defer L.mu.Unlock()
	L.head = nil
	L.tail = nil
	L.size = 0
	L.pos_map = make(map[string]*Node)
}

func (L *UniqueCLL) GetSize() int {
	// L.mu.Lock()
	// defer L.mu.Unlock()
	return L.size
}

func (L *UniqueCLL) GetHead() string {
	// L.mu.Lock()
	// defer L.mu.Unlock()
	return L.head.key
}

func (L *UniqueCLL) GetNode(uuid string) *types.Node {
	node, pres := L.pos_map[uuid]
	if pres {
		return &node.nodeinfo
	}
	return nil
}

func (L *UniqueCLL) PushBack(node types.Node) {
	// L.mu.Lock()
	// defer L.mu.Unlock()
	key := node.Uuid.String()
	_, pres := L.pos_map[key]
	if pres {
		return
	}
	new_node := &Node{}
	new_node.SetDefaults()
	new_node.key = key
	new_node.nodeinfo = node

	if L.head == nil {
		L.head = new_node
		L.tail = new_node
		new_node.next = new_node
		new_node.prev = new_node
	} else {
		new_node.prev = L.tail
		L.tail.next = new_node
		new_node.next = L.head
		L.head.prev = new_node
		L.tail = new_node
	}
	L.size += 1
	L.pos_map[key] = new_node
}

func (L *UniqueCLL) RemoveNode(key string) {
	// L.mu.Lock()
	// defer L.mu.Unlock()
	rem_node, pres := L.pos_map[key]
	if !pres {
		return
	}
	if L.head == L.tail {
		if L.head != rem_node {
			panic("If CLL of size 1, the removed node has to be the head and the tail")
		}
		L.head = nil
		L.tail = nil
	} else if L.head == rem_node {
		L.tail.next = L.head.next
		L.head.next.prev = L.tail
		L.head = L.head.next
	} else if L.tail == rem_node {
		L.tail.prev.next = L.head
		L.head.prev = L.tail.prev
		L.tail = L.tail.prev
	} else {
		rem_node.prev.next = rem_node.next
		rem_node.next.prev = rem_node.prev
	}
	L.size -= 1
	delete(L.pos_map, key)
}

// Gets next and previous element
func (L *UniqueCLL) GetNeighbors(key string) []types.Node {
	// L.mu.Lock()
	var machines []types.Node
	curr_node, pres := L.pos_map[key]
	if !pres {
		// L.mu.Unlock()
		fmt.Printf("[GetNeighbors] id: %s not found\n", key)
		return machines
	}
	if L.size == 2 {
		machines = append(machines, curr_node.next.nodeinfo)
	} else if L.size > 2 {
		machines = append(machines, curr_node.prev.nodeinfo)
		machines = append(machines, curr_node.next.nodeinfo)
	}
	// L.mu.Unlock()
	return machines
}

// returns list of node IPs for each node in the ring
func (L *UniqueCLL) GetIPList() []string {
	// L.mu.Lock()
	// defer L.mu.Unlock()
	var machine_ids []string
	curr_node := L.head
	for i := 0; i < L.size; i++ {
		machine_ids = append(machine_ids, curr_node.nodeinfo.Ip)
		curr_node = curr_node.next
	}
	return machine_ids
}

// gets the nodeinfo for each node in the ring
func (L *UniqueCLL) GetNodes(onlyRiders bool) []types.Node {
	var machine_ids []types.Node
	curr_node := L.head
	for i := 0; i < L.size; i++ {
		if onlyRiders { // only add node to list if it is a rider node
			if curr_node.nodeinfo.NodeType == types.Rider {
				machine_ids = append(machine_ids, curr_node.nodeinfo)
			}
		} else { // if onlyRiders is disabled just add all nodes
			machine_ids = append(machine_ids, curr_node.nodeinfo)
		}
		curr_node = curr_node.next
	}
	return machine_ids
}

func (L *UniqueCLL) PrintList() {
	machine_names := L.GetNodes(false)
	for i := 0; i < len(machine_names); i++ {
		fmt.Println(machine_names[i].Uuid.String())
	}
}
