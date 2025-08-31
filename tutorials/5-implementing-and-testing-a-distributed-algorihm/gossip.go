package gossip

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	mathrand "math/rand"
	"sync"
	"time"
)

// GossipMessage represents a piece of information spreading through the network
type GossipMessage struct {
	ID        string    `json:"id"`        // unique message identifier
	Content   string    `json:"content"`   // the actual information
	Sender    int       `json:"sender"`    // original sender node id
	Timestamp time.Time `json:"timestamp"` // when message was created
	TTL       int       `json:"ttl"`       // time-to-live (hops remaining)
}

// GossipNode represents a node in the gossip network
type GossipNode struct {
	id           int
	addr         Address
	peers        []Address // known peer addresses
	node         *Node
	seenMessages map[string]bool // prevent message loops
	receivedMsgs []GossipMessage // messages this node has received
	mu           sync.RWMutex

	// statistics
	messagesSent     int
	messagesReceived int
}

// NewGossipNode creates a new gossip node
func NewGossipNode(network Network, id int, port int) (*GossipNode, error) {
	addr := Address{IP: "127.0.0.1", Port: port}
	node, err := NewNode(network, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create gossip node %d: %v", id, err)
	}

	gossipnode := &GossipNode{
		id:           id,
		addr:         addr,
		peers:        make([]Address, 0),
		node:         node,
		seenMessages: make(map[string]bool),
		receivedMsgs: make([]GossipMessage, 0),
	}

	// set up message handlers
	gossipnode.SetupHandlers()

	return gossipnode, nil
}

func (gn *GossipNode) SetupHandlers() {
	// handle gossip messages
	gn.node.Handle("gossip", func(msg Message) error {
		var gossipmsg GossipMessage
		if err := json.Unmarshal(msg.Payload[7:], &gossipmsg); err != nil { // skip "gossip:" prefix
			return fmt.Errorf("failed to unmarshal gossip message: %v", err)
		}

		// Extract the immediate sender's node ID from the port
		immediateForwarder := msg.From.Port - 8000
		return gn.HandleGossipMessage(gossipmsg, immediateForwarder)
	})

	// handle peer discovery
	gn.node.Handle("discover", func(msg Message) error {
		// send back our peer list
		peerdata, _ := json.Marshal(gn.peers)
		return gn.node.Send(msg.From, "peers", peerdata)
	})
}

// AddPeer adds a peer to this node's peer list
func (gn *GossipNode) AddPeer(peeraddr Address) {
	gn.mu.Lock()
	defer gn.mu.Unlock()

	// don't add ourselves or duplicates
	if peeraddr.Port == gn.addr.Port {
		return
	}

	for _, existing := range gn.peers {
		if existing.Port == peeraddr.Port {
			return // already exists
		}
	}

	gn.peers = append(gn.peers, peeraddr)
}

// Start begins the node's operation
func (gn *GossipNode) Start() {
	gn.node.Start()
}

// Gossip initiates spreading of a new message
func (gn *GossipNode) Gossip(content string) error {
	// create unique message id
	msgid := gn.GenerateMessageID()

	gossipmsg := GossipMessage{
		ID:        msgid,
		Content:   content,
		Sender:    gn.id,
		Timestamp: time.Now(),
		TTL:       20, // maximum 20 hops
	}

	fmt.Printf("node %d starting gossip: '%s'\n", gn.id, content)

	return gn.SpreadGossip(gossipmsg)
}

func (gn *GossipNode) HandleGossipMessage(msg GossipMessage, immediateForwarder int) error {
	gn.mu.Lock()

	// check if we've seen this message before
	if gn.seenMessages[msg.ID] {
		gn.mu.Unlock()
		return nil // already processed
	}

	// mark as seen
	gn.seenMessages[msg.ID] = true
	gn.receivedMsgs = append(gn.receivedMsgs, msg)
	gn.messagesReceived++

	gn.mu.Unlock()

	if msg.Sender == immediateForwarder {
		// Direct from original sender
		fmt.Printf("node %d received gossip from node %d: '%s'\n", gn.id, msg.Sender, msg.Content)
	} else {
		// Forwarded by intermediate node
		fmt.Printf("node %d received gossip from node %d (via node %d): '%s'\n", gn.id, msg.Sender, immediateForwarder, msg.Content)
	}

	// decrease ttl and forward if still valid
	if msg.TTL > 0 {
		msg.TTL--
		go gn.SpreadGossip(msg)
	}

	return nil
}

func (gn *GossipNode) SpreadGossip(msg GossipMessage) error {
	gn.mu.RLock()
	peers := make([]Address, len(gn.peers))
	copy(peers, gn.peers)
	gn.mu.RUnlock()

	// send to all peers
	for _, peeraddr := range peers {
		go func(addr Address) {
			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("failed to marshal gossip message: %v", err)
				return
			}

			if err := gn.node.Send(addr, "gossip", data); err != nil {
				// peer might be down or partitioned - that's ok in gossip protocols
				return
			}

			gn.mu.Lock()
			gn.messagesSent++
			gn.mu.Unlock()
		}(peeraddr)
	}

	return nil
}

func (gn *GossipNode) GenerateMessageID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GetID returns the node's ID
func (gn *GossipNode) GetID() int {
	return gn.id
}

// GetStats returns node statistics
func (gn *GossipNode) GetStats() (int, int, int, int) {
	gn.mu.RLock()
	defer gn.mu.RUnlock()

	return len(gn.peers), len(gn.receivedMsgs), gn.messagesSent, gn.messagesReceived
}

// GetReceivedMessages returns all messages this node has received
func (gn *GossipNode) GetReceivedMessages() []GossipMessage {
	gn.mu.RLock()
	defer gn.mu.RUnlock()

	messages := make([]GossipMessage, len(gn.receivedMsgs))
	copy(messages, gn.receivedMsgs)
	return messages
}

// Close shuts down the node
func (gn *GossipNode) Close() error {
	return gn.node.Close()
}

// NetworkBuilder helps create a network of gossip nodes with random topology
type NetworkBuilder struct {
	network Network
	nodes   []*GossipNode
}

func NewNetworkBuilder(network Network) *NetworkBuilder {
	return &NetworkBuilder{
		network: network,
		nodes:   make([]*GossipNode, 0),
	}
}

// CreateNodes creates the specified number of gossip nodes
func (nb *NetworkBuilder) CreateNodes(count int) error {
	fmt.Printf("creating %d gossip nodes...\n", count)

	for i := 0; i < count; i++ {
		node, err := NewGossipNode(nb.network, i, 8000+i)
		if err != nil {
			return fmt.Errorf("failed to create node %d: %v", i, err)
		}
		nb.nodes = append(nb.nodes, node)
	}

	return nil
}

// BuildRandomTopology creates random connections between nodes
func (nb *NetworkBuilder) BuildRandomTopology(peerspernode int) {
	fmt.Printf("building random topology (%d peers per node)...\n", peerspernode)

	for _, node := range nb.nodes {
		// randomly select peers for this node
		selectedpeers := nb.SelectRandomPeers(node.id, peerspernode)

		for _, peerid := range selectedpeers {
			peeraddr := Address{IP: "127.0.0.1", Port: 8000 + peerid}
			node.AddPeer(peeraddr)
		}
	}
}

func (nb *NetworkBuilder) SelectRandomPeers(nodeid int, count int) []int {
	peers := make([]int, 0)
	maxattempts := count * 3 // prevent infinite loop

	for len(peers) < count && maxattempts > 0 {
		candidate := mathrand.Intn(len(nb.nodes))
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

// StartAllNodes starts all nodes in the network
func (nb *NetworkBuilder) StartAllNodes() {
	fmt.Printf("starting %d nodes...\n", len(nb.nodes))

	for _, node := range nb.nodes {
		node.Start()
	}

	// give nodes time to start
	time.Sleep(100 * time.Millisecond)
}

// InitiateGossip starts gossip from a random node
func (nb *NetworkBuilder) InitiateGossip(content string) {
	if len(nb.nodes) == 0 {
		return
	}

	// pick a random node to start the gossip
	starter := mathrand.Intn(len(nb.nodes))
	nb.nodes[starter].Gossip(content)
}

// GetNodes returns all nodes in the network
func (nb *NetworkBuilder) GetNodes() []*GossipNode {
	return nb.nodes
}

// CloseAllNodes shuts down all nodes
func (nb *NetworkBuilder) CloseAllNodes() {
	for _, node := range nb.nodes {
		node.Close()
	}
}
