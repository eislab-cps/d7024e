# Introduction 
This tutorial demonstrates a **Gossip Protocol** (also called epidemic protocol) using 1000 nodes where each node randomly knows only a few other nodes. We'll see how information spreads through the network like a rumor or virus.

## What is a Gossip Protocol?
Gossip protocols mimic how rumors spread in human social networks:
- Each person (node) knows only a few other people
- When someone learns something new, they tell their friends
- Friends then tell their friends, and so on
- Eventually, almost everyone knows the information

This is used in real systems like:
- **Bitcoin**: Transaction propagation
- **Cassandra**: Database replication
- **Consul**: Service discovery
- **Kubernetes**: Cluster state updates

## The Algorithm
1. **Random Topology**: Each node randomly connects to 3-5 other nodes (its "peers")
2. **Gossip Spreading**: When a node receives new information, it forwards it to all its peers
3. **Duplicate Detection**: Nodes track what they've already seen to avoid infinite loops
4. **Probabilistic Coverage**: Not guaranteed to reach everyone, but reaches most nodes

## Implementation
```go
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

// gossipmessage represents a piece of information spreading through the network
type gossipmessage struct {
	id        string    `json:"id"`         // unique message identifier
	content   string    `json:"content"`    // the actual information
	sender    int       `json:"sender"`     // original sender node id
	timestamp time.time `json:"timestamp"`  // when message was created
	ttl       int       `json:"ttl"`        // time-to-live (hops remaining)
}

// gossipnode represents a node in the gossip network
type gossipnode struct {
	id           int
	addr         address
	peers        []address           // known peer addresses
	node         *node
	seenmessages map[string]bool     // prevent message loops
	receivedmsgs []gossipmessage     // messages this node has received
	mu           sync.rwmutex
	
	// statistics
	messagessent     int
	messagesreceived int
}

// newgossipnode creates a new gossip node
func newgossipnode(network network, id int, port int) (*gossipnode, error) {
	addr := address{ip: "127.0.0.1", port: port}
	node, err := newnode(network, addr)
	if err != nil {
		return nil, fmt.errorf("failed to create gossip node %d: %v", id, err)
	}

	gossipnode := &gossipnode{
		id:           id,
		addr:         addr,
		peers:        make([]address, 0),
		node:         node,
		seenmessages: make(map[string]bool),
		receivedmsgs: make([]gossipmessage, 0),
	}

	// set up message handlers
	gossipnode.setuphandlers()
	
	return gossipnode, nil
}

func (gn *gossipnode) setuphandlers() {
	// handle gossip messages
	gn.node.handle("gossip", func(msg message) error {
		var gossipmsg gossipmessage
		if err := json.unmarshal(msg.payload[7:], &gossipmsg); err != nil { // skip "gossip:" prefix
			return fmt.errorf("failed to unmarshal gossip message: %v", err)
		}
		
		return gn.handlegossipmessage(gossipmsg)
	})
	
	// handle peer discovery
	gn.node.handle("discover", func(msg message) error {
		// send back our peer list
		peerdata, _ := json.marshal(gn.peers)
		return gn.node.send(msg.from, "peers", peerdata)
	})
}

// addpeer adds a peer to this node's peer list
func (gn *gossipnode) addpeer(peeraddr address) {
	gn.mu.lock()
	defer gn.mu.unlock()
	
	// don't add ourselves or duplicates
	if peeraddr.port == gn.addr.port {
		return
	}
	
	for _, existing := range gn.peers {
		if existing.port == peeraddr.port {
			return // already exists
		}
	}
	
	gn.peers = append(gn.peers, peeraddr)
}

// start begins the node's operation
func (gn *gossipnode) start() {
	gn.node.start()
}

// gossip initiates spreading of a new message
func (gn *gossipnode) gossip(content string) error {
	// create unique message id
	msgid := gn.generatemessageid()
	
	gossipmsg := gossipmessage{
		id:        msgid,
		content:   content,
		sender:    gn.id,
		timestamp: time.now(),
		ttl:       20, // maximum 20 hops
	}
	
	fmt.printf("node %d starting gossip: '%s'\n", gn.id, content)
	
	return gn.spreadgossip(gossipmsg)
}

func (gn *gossipnode) handlegossipmessage(msg gossipmessage) error {
	gn.mu.lock()
	
	// check if we've seen this message before
	if gn.seenmessages[msg.id] {
		gn.mu.unlock()
		return nil // already processed
	}
	
	// mark as seen
	gn.seenmessages[msg.id] = true
	gn.receivedmsgs = append(gn.receivedmsgs, msg)
	gn.messagesreceived++
	
	gn.mu.unlock()
	
	fmt.printf("node %d received gossip from node %d: '%s'\n", gn.id, msg.sender, msg.content)
	
	// decrease ttl and forward if still valid
	if msg.ttl > 0 {
		msg.ttl--
		go gn.spreadgossip(msg)
	}
	
	return nil
}

func (gn *gossipnode) spreadgossip(msg gossipmessage) error {
	gn.mu.rlock()
	peers := make([]address, len(gn.peers))
	copy(peers, gn.peers)
	gn.mu.runlock()
	
	// send to all peers
	for _, peeraddr := range peers {
		go func(addr address) {
			data, err := json.marshal(msg)
			if err != nil {
				log.printf("failed to marshal gossip message: %v", err)
				return
			}
			
			if err := gn.node.send(addr, "gossip", data); err != nil {
				// peer might be down or partitioned - that's ok in gossip protocols
				return
			}
			
			gn.mu.lock()
			gn.messagessent++
			gn.mu.unlock()
		}(peeraddr)
	}
	
	return nil
}

func (gn *gossipnode) generatemessageid() string {
	bytes := make([]byte, 16)
	rand.read(bytes)
	return hex.encodetostring(bytes)
}

// getstats returns node statistics
func (gn *gossipnode) getstats() (int, int, int, int) {
	gn.mu.rlock()
	defer gn.mu.runlock()
	
	return len(gn.peers), len(gn.receivedmsgs), gn.messagessent, gn.messagesreceived
}

// getreceivedmessages returns all messages this node has received
func (gn *gossipnode) getreceivedmessages() []gossipmessage {
	gn.mu.rlock()
	defer gn.mu.runlock()
	
	messages := make([]gossipmessage, len(gn.receivedmsgs))
	copy(messages, gn.receivedmsgs)
	return messages
}

// close shuts down the node
func (gn *gossipnode) close() error {
	return gn.node.close()
}

// networkbuilder helps create a network of gossip nodes with random topology
type networkbuilder struct {
	network network
	nodes   []*gossipnode
}

func newnetworkbuilder(network network) *networkbuilder {
	return &networkbuilder{
		network: network,
		nodes:   make([]*gossipnode, 0),
	}
}

// createnodes creates the specified number of gossip nodes
func (nb *networkbuilder) createnodes(count int) error {
	fmt.printf("creating %d gossip nodes...\n", count)
	
	for i := 0; i < count; i++ {
		node, err := newgossipnode(nb.network, i, 8000+i)
		if err != nil {
			return fmt.errorf("failed to create node %d: %v", i, err)
		}
		nb.nodes = append(nb.nodes, node)
	}
	
	return nil
}

// buildrandomtopology creates random connections between nodes
func (nb *networkbuilder) buildrandomtopology(peerspernode int) {
	fmt.printf("building random topology (%d peers per node)...\n", peerspernode)
	
	for _, node := range nb.nodes {
		// randomly select peers for this node
		selectedpeers := nb.selectrandompeers(node.id, peerspernode)
		
		for _, peerid := range selectedpeers {
			peeraddr := address{ip: "127.0.0.1", port: 8000 + peerid}
			node.addpeer(peeraddr)
		}
	}
}

func (nb *networkbuilder) selectrandompeers(nodeid int, count int) []int {
	peers := make([]int, 0)
	maxattempts := count * 3 // prevent infinite loop
	
	for len(peers) < count && maxattempts > 0 {
		candidate := rand.intn(len(nb.nodes))
		if candidate == nodeid {
			maxattempts--
			continue // don't add ourselves
		}
		
		// check if already selected
		found := false
		for _, existing := range peers {
			if existing == candidate {
				found = true
				break
			}
		}
		
		if !found {
			peers = append(peers, candidate)
		}
		maxattempts--
	}
	
	return peers
}

// startallnodes starts all nodes in the network
func (nb *networkbuilder) startallnodes() {
	fmt.printf("starting %d nodes...\n", len(nb.nodes))
	
	for _, node := range nb.nodes {
		node.start()
	}
	
	// give nodes time to start
	time.sleep(100 * time.millisecond)
}

// initiategossip starts gossip from a random node
func (nb *networkbuilder) initiategossip(content string) {
	if len(nb.nodes) == 0 {
		return
	}
	
	// pick a random node to start the gossip
	starter := rand.intn(len(nb.nodes))
	nb.nodes[starter].gossip(content)
}

// getnodes returns all nodes in the network
func (nb *networkbuilder) getnodes() []*gossipnode {
	return nb.nodes
}

// closeallnodes shuts down all nodes
func (nb *networkbuilder) closeallnodes() {
	for _, node := range nb.nodes {
		node.close()
	}
}
```

## Example Usage
```go
func TestGossipProtocol1000Nodes(t *testing.T) {
	// Create network
	network := NewMockNetwork()
	builder := NewNetworkBuilder(network)
	
	// Build network with 1000 nodes, each knowing 4 random peers
	err := builder.CreateNodes(1000)
	if err != nil {
		t.Fatal(err)
	}
	
	builder.BuildRandomTopology(4)
	builder.StartAllNodes()
	
	// Start gossip from random node
	builder.InitiateGossip("Hello from the gossip network!")
	
	// Wait for propagation
	time.Sleep(2 * time.Second)
	
	// Analyze results
	nodes := builder.GetNodes()
	totalReached := 0
	totalMessagesSent := 0
	
	for _, node := range nodes {
		peers, received, sent, _ := node.GetStats()
		if received > 0 {
			totalReached++
		}
		totalMessagesSent += sent
		
		// Print sample of nodes that received the message
		if totalReached <= 10 && received > 0 {
			fmt.Printf("Node %d (peers: %d) received %d messages\n", 
				node.id, peers, received)
		}
	}
	
	fmt.Printf("\nGossip Results:")
	fmt.Printf("- Network size: %d nodes\n", len(nodes))
	fmt.Printf("- Nodes reached: %d (%.1f%%)\n", 
		totalReached, float64(totalReached)/float64(len(nodes))*100)
	fmt.Printf("- Total messages sent: %d\n", totalMessagesSent)
	fmt.Printf("- Average messages per node: %.1f\n", 
		float64(totalMessagesSent)/float64(len(nodes)))
	
	builder.CloseAllNodes()
}
```

## What Students Will Observe

**Rapid Propagation**: Information spreads quickly through the network despite each node knowing only a few others.

**High Coverage**: Most nodes (90%+ typically) receive the message even without guaranteed delivery.

**Network Efficiency**: The gossip spreads with manageable message overhead.

**Fault Tolerance**: Even if some nodes are down or partitioned, the gossip continues spreading through alternative paths.

## Key Learning Points

- **Emergent Behavior**: Simple local rules create complex global behavior
- **Trade-offs**: High availability vs. guaranteed delivery
- **Scalability**: Works with thousands of nodes
- **Real-world Applications**: Understanding how P2P systems work

This demonstrates fundamental distributed systems concepts while being engaging and practical for students to experiment with.
