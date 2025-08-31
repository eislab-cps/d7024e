package gossip

import (
	"fmt"
	"testing"
	"time"
)

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
				node.GetID(), peers, received)
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
