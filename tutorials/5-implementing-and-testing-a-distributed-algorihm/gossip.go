package gossip

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	mathrand "math/rand"
	"os"
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

// NetworkTopology represents the network structure for visualization
type NetworkTopology struct {
	Nodes    []NodeInfo    `json:"nodes"`
	Edges    []EdgeInfo    `json:"edges"`
	Clusters []ClusterInfo `json:"clusters"`
}

// ClusterInfo represents a connected component cluster
type ClusterInfo struct {
	ID       int   `json:"id"`
	NodeIDs  []int `json:"nodeIds"`
	Size     int   `json:"size"`
	CenterX  int   `json:"centerX"`
	CenterY  int   `json:"centerY"`
	IsIsolated bool `json:"isIsolated"`
}

// NodeInfo represents a node in the visualization
type NodeInfo struct {
	ID        int    `json:"id"`
	Addr      string `json:"addr"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	ClusterID int    `json:"clusterId"`
}

// EdgeInfo represents a connection between nodes
type EdgeInfo struct {
	From int `json:"from"`
	To   int `json:"to"`
}

// MessageTrace represents a single message transmission event
type MessageTrace struct {
	Timestamp         time.Time `json:"timestamp"`
	MessageID         string    `json:"messageId"`
	OriginalSender    int       `json:"originalSender"`
	ImmediateForwarder int      `json:"immediateForwarder"`
	Receiver          int       `json:"receiver"`
	Content           string    `json:"content"`
	TTL               int       `json:"ttl"`
	IsDirect          bool      `json:"isDirect"`
}

// VisualizationData contains all data needed for visualization
type VisualizationData struct {
	Topology NetworkTopology `json:"topology"`
	Traces   []MessageTrace  `json:"traces"`
	StartTime time.Time      `json:"startTime"`
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
	
	// visualization tracking
	builder      *NetworkBuilder // reference to builder for trace logging

	// statistics
	messagesSent     int
	messagesReceived int
}

// NewGossipNode creates a new gossip node
func NewGossipNode(network Network, id int, port int, builder *NetworkBuilder) (*GossipNode, error) {
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
		builder:      builder,
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

	// Log message trace for visualization
	if gn.builder != nil {
		trace := MessageTrace{
			Timestamp:         time.Now(),
			MessageID:         msg.ID,
			OriginalSender:    msg.Sender,
			ImmediateForwarder: immediateForwarder,
			Receiver:          gn.id,
			Content:           msg.Content,
			TTL:               msg.TTL,
			IsDirect:          msg.Sender == immediateForwarder,
		}
		gn.builder.traceMu.Lock()
		gn.builder.traces = append(gn.builder.traces, trace)
		gn.builder.traceMu.Unlock()
	}

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
	traces  []MessageTrace
	startTime time.Time
	traceMu sync.Mutex
}

func NewNetworkBuilder(network Network) *NetworkBuilder {
	return &NetworkBuilder{
		network: network,
		nodes:   make([]*GossipNode, 0),
		traces:  make([]MessageTrace, 0),
		startTime: time.Now(),
	}
}

// CreateNodes creates the specified number of gossip nodes
func (nb *NetworkBuilder) CreateNodes(count int) error {
	fmt.Printf("creating %d gossip nodes...\n", count)

	for i := 0; i < count; i++ {
		node, err := NewGossipNode(nb.network, i, 8000+i, nb)
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

// ExportVisualizationData exports network topology and message traces to JSON files
func (nb *NetworkBuilder) ExportVisualizationData(outputDir string) error {
	// Create output directory if it doesn't exist
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Generate network topology
	topology := nb.generateTopology()

	// Create visualization data
	visData := VisualizationData{
		Topology:  topology,
		Traces:    nb.traces,
		StartTime: nb.startTime,
	}

	// Write to JSON file
	data, err := json.MarshalIndent(visData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal visualization data: %v", err)
	}

	filename := fmt.Sprintf("%s/network_visualization.json", outputDir)
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write visualization file: %v", err)
	}

	fmt.Printf("Exported visualization data to %s\n", filename)
	fmt.Printf("Total nodes: %d, Total message traces: %d\n", len(topology.Nodes), len(nb.traces))
	return nil
}

// generateTopology creates the network topology for visualization
func (nb *NetworkBuilder) generateTopology() NetworkTopology {
	nodes := make([]NodeInfo, len(nb.nodes))
	edges := make([]EdgeInfo, 0)

	// Create bidirectional adjacency map for proper connectivity analysis
	nodeConnections := make(map[int][]int)
	for i := 0; i < len(nb.nodes); i++ {
		nodeConnections[i] = make([]int, 0)
	}

	// Build edges and bidirectional connections
	for _, node := range nb.nodes {
		node.mu.RLock()
		for _, peerAddr := range node.peers {
			peerID := peerAddr.Port - 8000
			if peerID >= 0 && peerID < len(nb.nodes) {
				// Add edge for visualization
				edges = append(edges, EdgeInfo{
					From: node.GetID(),
					To:   peerID,
				})
				// Add bidirectional connection for connectivity analysis
				nodeConnections[node.GetID()] = append(nodeConnections[node.GetID()], peerID)
				nodeConnections[peerID] = append(nodeConnections[peerID], node.GetID())
			}
		}
		node.mu.RUnlock()
	}

	// Remove duplicates from connections
	for nodeID := range nodeConnections {
		seen := make(map[int]bool)
		unique := make([]int, 0)
		for _, conn := range nodeConnections[nodeID] {
			if !seen[conn] {
				seen[conn] = true
				unique = append(unique, conn)
			}
		}
		nodeConnections[nodeID] = unique
	}

	// Find connected components (clusters) first
	clusters := nb.findConnectedComponents(nodeConnections)

	// Use specialized layout that separates islands
	positions := nb.layoutWithIslands(nodeConnections, clusters, 1200, 800)

	// Create nodes with computed positions and cluster assignments
	for i, node := range nb.nodes {
		pos := positions[node.GetID()]
		clusterID := nb.findNodeCluster(node.GetID(), clusters)
		nodes[i] = NodeInfo{
			ID:        node.GetID(),
			Addr:      node.addr.String(),
			X:         int(pos.X),
			Y:         int(pos.Y),
			ClusterID: clusterID,
		}
	}

	// Generate cluster info with statistics
	clusterInfos := nb.generateClusterInfo(clusters, positions)

	return NetworkTopology{
		Nodes:    nodes,
		Edges:    edges,
		Clusters: clusterInfos,
	}
}

// layoutWithIslands creates a layout that separates isolated components
func (nb *NetworkBuilder) layoutWithIslands(connections map[int][]int, clusters [][]int, width, height int) map[int]Position {
	positions := make(map[int]Position)
	
	// Find the largest connected component
	largestCluster := 0
	largestSize := 0
	for i, cluster := range clusters {
		if len(cluster) > largestSize {
			largestSize = len(cluster)
			largestCluster = i
		}
	}
	
	// Layout main component on the right side (60% of width)
	mainWidth := int(float64(width) * 0.6)
	mainStartX := width - mainWidth
	if len(clusters[largestCluster]) > 0 {
		mainPositions := nb.forceDirectedLayoutForCluster(clusters[largestCluster], connections, mainWidth, height)
		for _, nodeID := range clusters[largestCluster] {
			pos := mainPositions[nodeID]
			positions[nodeID] = Position{
				X: pos.X + float64(mainStartX),
				Y: pos.Y,
			}
		}
	}
	
	// Layout isolated components on the left side (40% of width)
	isolatedWidth := width - mainWidth - 50 // 50px padding
	isolatedClusters := make([][]int, 0)
	for i, cluster := range clusters {
		if i != largestCluster {
			isolatedClusters = append(isolatedClusters, cluster)
		}
	}
	
	if len(isolatedClusters) > 0 {
		// Arrange isolated clusters in a grid on the left
		cols := 4 // 4 columns for isolated clusters
		rows := (len(isolatedClusters) + cols - 1) / cols
		
		clusterWidth := isolatedWidth / cols
		clusterHeight := height / rows
		
		for i, cluster := range isolatedClusters {
			col := i % cols
			row := i / cols
			
			clusterX := col * clusterWidth + 25 // 25px padding
			clusterY := row * clusterHeight + 25
			clusterW := clusterWidth - 50
			clusterH := clusterHeight - 50
			
			if len(cluster) == 1 {
				// Single node - place in center of its area
				positions[cluster[0]] = Position{
					X: float64(clusterX + clusterW/2),
					Y: float64(clusterY + clusterH/2),
				}
			} else {
				// Multiple nodes - use mini force-directed layout
				clusterPositions := nb.forceDirectedLayoutForCluster(cluster, connections, clusterW, clusterH)
				for _, nodeID := range cluster {
					pos := clusterPositions[nodeID]
					positions[nodeID] = Position{
						X: pos.X + float64(clusterX),
						Y: pos.Y + float64(clusterY),
					}
				}
			}
		}
	}
	
	return positions
}

// forceDirectedLayoutForCluster runs force-directed layout for a specific cluster
func (nb *NetworkBuilder) forceDirectedLayoutForCluster(cluster []int, connections map[int][]int, width, height int) map[int]Position {
	positions := make(map[int]Position)
	velocities := make(map[int]Position)
	
	// Initialize random positions within bounds
	for _, nodeID := range cluster {
		positions[nodeID] = Position{
			X: mathrand.Float64() * float64(width),
			Y: mathrand.Float64() * float64(height),
		}
		velocities[nodeID] = Position{X: 0, Y: 0}
	}
	
	// Run simulation for fewer iterations since clusters are smaller
	iterations := 200
	for iter := 0; iter < iterations; iter++ {
		forces := make(map[int]Position)
		for _, nodeID := range cluster {
			forces[nodeID] = Position{X: 0, Y: 0}
		}
		
		// Repulsion forces between all nodes in cluster
		for i, nodeA := range cluster {
			for j, nodeB := range cluster {
				if i >= j {
					continue
				}
				
				posA := positions[nodeA]
				posB := positions[nodeB]
				
				dx := posA.X - posB.X
				dy := posA.Y - posB.Y
				dist := math.Sqrt(dx*dx + dy*dy)
				
				if dist < 1 {
					dist = 1
				}
				
				repulsionForce := 500.0 / (dist * dist)
				fx := (dx / dist) * repulsionForce
				fy := (dy / dist) * repulsionForce
				
				forces[nodeA] = Position{
					X: forces[nodeA].X + fx,
					Y: forces[nodeA].Y + fy,
				}
				forces[nodeB] = Position{
					X: forces[nodeB].X - fx,
					Y: forces[nodeB].Y - fy,
				}
			}
		}
		
		// Attraction forces for connected nodes
		for _, nodeA := range cluster {
			for _, nodeB := range connections[nodeA] {
				// Only consider connections within this cluster
				inCluster := false
				for _, clusterNode := range cluster {
					if clusterNode == nodeB {
						inCluster = true
						break
					}
				}
				if !inCluster {
					continue
				}
				
				posA := positions[nodeA]
				posB := positions[nodeB]
				
				dx := posB.X - posA.X
				dy := posB.Y - posA.Y
				dist := math.Sqrt(dx*dx + dy*dy)
				
				if dist > 0 {
					attractionForce := 0.1 * dist
					fx := (dx / dist) * attractionForce
					fy := (dy / dist) * attractionForce
					
					forces[nodeA] = Position{
						X: forces[nodeA].X + fx,
						Y: forces[nodeA].Y + fy,
					}
				}
			}
		}
		
		// Update positions with damping
		damping := 0.9
		for _, nodeID := range cluster {
			velocities[nodeID] = Position{
				X: velocities[nodeID].X*damping + forces[nodeID].X*0.01,
				Y: velocities[nodeID].Y*damping + forces[nodeID].Y*0.01,
			}
			
			positions[nodeID] = Position{
				X: positions[nodeID].X + velocities[nodeID].X,
				Y: positions[nodeID].Y + velocities[nodeID].Y,
			}
			
			// Keep within bounds
			if positions[nodeID].X < 10 {
				positions[nodeID] = Position{X: 10, Y: positions[nodeID].Y}
			}
			if positions[nodeID].X > float64(width-10) {
				positions[nodeID] = Position{X: float64(width - 10), Y: positions[nodeID].Y}
			}
			if positions[nodeID].Y < 10 {
				positions[nodeID] = Position{X: positions[nodeID].X, Y: 10}
			}
			if positions[nodeID].Y > float64(height-10) {
				positions[nodeID] = Position{X: positions[nodeID].X, Y: float64(height - 10)}
			}
		}
	}
	
	return positions
}

// Position represents a 2D position
type Position struct {
	X, Y float64
}

// simulateForceDirectedLayout computes better node positions using force simulation
func (nb *NetworkBuilder) simulateForceDirectedLayout(connections map[int][]int, width, height float64) map[int]Position {
	positions := make(map[int]Position)
	velocities := make(map[int]Position)
	
	// Initialize random positions
	for _, node := range nb.nodes {
		positions[node.GetID()] = Position{
			X: mathrand.Float64() * width,
			Y: mathrand.Float64() * height,
		}
		velocities[node.GetID()] = Position{X: 0, Y: 0}
	}

	// Force simulation parameters
	const (
		iterations = 300
		repulsion = 5000.0    // Repulsive force between all nodes
		attraction = 0.1      // Attractive force between connected nodes  
		damping = 0.9         // Velocity damping
		minDistance = 50.0    // Minimum distance between nodes
	)

	// Run simulation
	for iter := 0; iter < iterations; iter++ {
		forces := make(map[int]Position)
		
		// Initialize forces
		for _, node := range nb.nodes {
			forces[node.GetID()] = Position{X: 0, Y: 0}
		}

		// Repulsive forces (all nodes repel each other)
		for _, node1 := range nb.nodes {
			for _, node2 := range nb.nodes {
				if node1.GetID() == node2.GetID() {
					continue
				}

				pos1 := positions[node1.GetID()]
				pos2 := positions[node2.GetID()]
				
				dx := pos1.X - pos2.X
				dy := pos1.Y - pos2.Y
				distance := math.Sqrt(dx*dx + dy*dy)
				
				if distance < minDistance {
					distance = minDistance
				}
				
				force := repulsion / (distance * distance)
				fx := force * dx / distance
				fy := force * dy / distance
				
				f1 := forces[node1.GetID()]
				f1.X += fx
				f1.Y += fy
				forces[node1.GetID()] = f1
			}
		}

		// Attractive forces (connected nodes attract)
		for nodeID, peers := range connections {
			for _, peerID := range peers {
				pos1 := positions[nodeID]
				pos2 := positions[peerID]
				
				dx := pos2.X - pos1.X
				dy := pos2.Y - pos1.Y
				distance := math.Sqrt(dx*dx + dy*dy)
				
				if distance > 0 {
					force := attraction * distance
					fx := force * dx / distance
					fy := force * dy / distance
					
					f1 := forces[nodeID]
					f1.X += fx
					f1.Y += fy
					forces[nodeID] = f1
				}
			}
		}

		// Apply forces and update positions
		for _, node := range nb.nodes {
			nodeID := node.GetID()
			
			// Update velocity
			vel := velocities[nodeID]
			force := forces[nodeID]
			vel.X = (vel.X + force.X) * damping
			vel.Y = (vel.Y + force.Y) * damping
			velocities[nodeID] = vel
			
			// Update position
			pos := positions[nodeID]
			pos.X += vel.X
			pos.Y += vel.Y
			
			// Keep within bounds
			if pos.X < 50 { pos.X = 50 }
			if pos.X > width-50 { pos.X = width-50 }
			if pos.Y < 50 { pos.Y = 50 }
			if pos.Y > height-50 { pos.Y = height-50 }
			
			positions[nodeID] = pos
		}
	}

	return positions
}

// findConnectedComponents finds all connected components in the graph using DFS
func (nb *NetworkBuilder) findConnectedComponents(connections map[int][]int) [][]int {
	visited := make(map[int]bool)
	var clusters [][]int

	// Create bidirectional adjacency list
	adjacency := make(map[int][]int)
	for nodeID, peers := range connections {
		if adjacency[nodeID] == nil {
			adjacency[nodeID] = make([]int, 0)
		}
		for _, peerID := range peers {
			adjacency[nodeID] = append(adjacency[nodeID], peerID)
			if adjacency[peerID] == nil {
				adjacency[peerID] = make([]int, 0)
			}
			adjacency[peerID] = append(adjacency[peerID], nodeID)
		}
	}

	// Ensure all nodes are in adjacency list (including isolated ones)
	for _, node := range nb.nodes {
		if adjacency[node.GetID()] == nil {
			adjacency[node.GetID()] = make([]int, 0)
		}
	}

	// DFS to find connected components
	var dfs func(nodeID int, component []int) []int
	dfs = func(nodeID int, component []int) []int {
		if visited[nodeID] {
			return component
		}
		visited[nodeID] = true
		component = append(component, nodeID)

		for _, neighbor := range adjacency[nodeID] {
			component = dfs(neighbor, component)
		}
		return component
	}

	// Find all connected components
	for _, node := range nb.nodes {
		nodeID := node.GetID()
		if !visited[nodeID] {
			component := dfs(nodeID, []int{})
			if len(component) > 0 {
				clusters = append(clusters, component)
			}
		}
	}

	return clusters
}

// findNodeCluster returns the cluster ID for a given node
func (nb *NetworkBuilder) findNodeCluster(nodeID int, clusters [][]int) int {
	for clusterID, cluster := range clusters {
		for _, id := range cluster {
			if id == nodeID {
				return clusterID
			}
		}
	}
	return -1 // Should never happen
}

// generateClusterInfo creates cluster information for visualization
func (nb *NetworkBuilder) generateClusterInfo(clusters [][]int, positions map[int]Position) []ClusterInfo {
	clusterInfos := make([]ClusterInfo, len(clusters))

	// Find the largest connected component
	largestCluster := 0
	largestSize := 0
	for i, cluster := range clusters {
		if len(cluster) > largestSize {
			largestSize = len(cluster)
			largestCluster = i
		}
	}

	for i, cluster := range clusters {
		// Calculate cluster center
		var totalX, totalY float64
		for _, nodeID := range cluster {
			pos := positions[nodeID]
			totalX += pos.X
			totalY += pos.Y
		}

		centerX := int(totalX / float64(len(cluster)))
		centerY := int(totalY / float64(len(cluster)))

		// Mark all components except the largest as isolated
		isIsolated := (i != largestCluster)

		clusterInfos[i] = ClusterInfo{
			ID:         i,
			NodeIDs:    cluster,
			Size:       len(cluster),
			CenterX:    centerX,
			CenterY:    centerY,
			IsIsolated: isIsolated,
		}
	}

	return clusterInfos
}
