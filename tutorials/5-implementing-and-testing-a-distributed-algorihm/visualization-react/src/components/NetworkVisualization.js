import React, { useEffect, useRef, useState, useCallback } from 'react';
import * as d3 from 'd3';
import './NetworkVisualization.css';

const NetworkVisualization = ({ data, currentTrace, replayProgress = 0, isPlaying = false }) => {
  const svgRef = useRef();
  const [selectedNode, setSelectedNode] = useState(null);
  const [highlightedNodes, setHighlightedNodes] = useState(new Set());
  const [messageAnimations, setMessageAnimations] = useState(new Map());
  const [nodeStates, setNodeStates] = useState(new Map()); // Track which nodes have received messages
  const [showClusters, setShowClusters] = useState(true);
  const [highlightIsolated, setHighlightIsolated] = useState(false);
  const [clusterColors] = useState(() => {
    // Generate distinct colors for clusters
    const colors = [
      '#ff6b6b', '#4ecdc4', '#45b7d1', '#96ceb4', '#feca57',
      '#ff9ff3', '#54a0ff', '#5f27cd', '#00d2d3', '#ff9f43',
      '#a55eea', '#26de81', '#fd79a8', '#fdcb6e', '#6c5ce7',
      '#fd79a8', '#a29bfe', '#6c5ce7', '#74b9ff', '#81ecec'
    ];
    return colors;
  });

  // Animate message propagation when currentTrace changes
  useEffect(() => {
    if (!currentTrace || !data?.topology) return;

    const { nodes } = data.topology;
    const sender = nodes.find(n => n.id === currentTrace.immediateForwarder);
    const receiver = nodes.find(n => n.id === currentTrace.receiver);
    
    if (!sender || !receiver) return;

    // Add message animation
    animateMessage(sender, receiver, currentTrace);
    
    // Update node state
    setNodeStates(prev => {
      const newState = new Map(prev);
      newState.set(currentTrace.receiver, {
        hasMessage: true,
        isOriginalSender: currentTrace.originalSender === currentTrace.receiver,
        lastReceived: Date.now()
      });
      return newState;
    });

  }, [currentTrace]);

  const animateMessage = useCallback((fromNode, toNode, trace) => {
    const svg = d3.select(svgRef.current);
    const container = svg.select('.main-container');
    
    if (container.empty()) return;

    // Create message particle
    const message = container.append('circle')
      .classed('message-particle', true)
      .attr('r', 4)
      .attr('fill', trace.isDirect ? '#ff6b6b' : '#4ecdc4')
      .attr('stroke', '#fff')
      .attr('stroke-width', 2)
      .attr('opacity', 0.9)
      .attr('cx', fromNode.x)
      .attr('cy', fromNode.y);

    // Animate along the connection
    message.transition()
      .duration(800)
      .ease(d3.easeQuadInOut)
      .attr('cx', toNode.x)
      .attr('cy', toNode.y)
      .on('end', () => {
        // Pulse effect on arrival
        const pulseCircle = container.append('circle')
          .attr('cx', toNode.x)
          .attr('cy', toNode.y)
          .attr('r', 6)
          .attr('fill', 'none')
          .attr('stroke', trace.isDirect ? '#ff6b6b' : '#4ecdc4')
          .attr('stroke-width', 3)
          .attr('opacity', 0.8);

        pulseCircle.transition()
          .duration(300)
          .attr('r', 20)
          .attr('opacity', 0)
          .on('end', () => pulseCircle.remove());
          
        message.remove();
      });
  }, []);

  useEffect(() => {
    if (!data || !data.topology) return;

    const svg = d3.select(svgRef.current);
    svg.selectAll("*").remove();

    const width = 1200;
    const height = 800;
    const { nodes, edges } = data.topology;

    svg.attr('width', width).attr('height', height);

    // Create scales for positioning
    const xExtent = d3.extent(nodes, d => d.x);
    const yExtent = d3.extent(nodes, d => d.y);
    
    const xScale = d3.scaleLinear()
      .domain(xExtent)
      .range([50, width - 50]);
    
    const yScale = d3.scaleLinear()
      .domain(yExtent)
      .range([50, height - 50]);

    // Create container for zoom
    const container = svg.append('g').classed('main-container', true);

    // Add zoom behavior
    const zoom = d3.zoom()
      .scaleExtent([0.1, 4])
      .on('zoom', (event) => {
        container.attr('transform', event.transform);
      });

    svg.call(zoom);

    // Add visual separator between islands and main component
    if (data?.topology?.clusters && data.topology.clusters.some(c => c.isIsolated)) {
      const separatorX = width * 0.4; // 40% from left
      container.append('line')
        .attr('x1', separatorX)
        .attr('y1', 0)
        .attr('x2', separatorX)
        .attr('y2', height)
        .attr('stroke', '#ddd')
        .attr('stroke-width', 2)
        .attr('stroke-dasharray', '5,5')
        .classed('island-separator', true);
        
      // Add labels
      container.append('text')
        .attr('x', separatorX / 2)
        .attr('y', 30)
        .attr('text-anchor', 'middle')
        .style('font-size', '14px')
        .style('font-weight', 'bold')
        .style('fill', '#666')
        .text('🏝️ Isolated Islands');
        
      container.append('text')
        .attr('x', separatorX + (width - separatorX) / 2)
        .attr('y', 30)
        .attr('text-anchor', 'middle')
        .style('font-size', '14px')
        .style('font-weight', 'bold')
        .style('fill', '#666')
        .text('🌐 Main Network');
    }

    // Add rounded backgrounds for nodes that didn't receive gossip messages
    if (data?.traces) {
      // Get set of nodes that received messages
      const nodesWithMessages = new Set();
      data.traces.forEach(trace => {
        nodesWithMessages.add(trace.receiver);
      });

      // Add light red rounded backgrounds for nodes that didn't receive messages
      nodes.forEach(node => {
        if (!nodesWithMessages.has(node.id)) {
          container.append('circle')
            .attr('cx', xScale(node.x))
            .attr('cy', yScale(node.y))
            .attr('r', 12)
            .attr('fill', '#ffebee')
            .attr('stroke', '#f44336')
            .attr('stroke-width', 1)
            .attr('stroke-dasharray', '3,2')
            .attr('opacity', 0.8)
            .classed('node-unreached-background', true);
        }
      });
    }

    // Define arrow marker for directed edges
    const defs = svg.append('defs');
    
    // Default arrow marker
    defs.append('marker')
      .attr('id', 'arrow')
      .attr('viewBox', '0 -5 10 10')
      .attr('refX', 8)
      .attr('refY', 0)
      .attr('markerWidth', 6)
      .attr('markerHeight', 6)
      .attr('orient', 'auto')
      .append('path')
      .attr('d', 'M0,-5L10,0L0,5')
      .attr('fill', '#999');
    
    // Active message arrow marker
    defs.append('marker')
      .attr('id', 'arrow-active')
      .attr('viewBox', '0 -5 10 10')
      .attr('refX', 8)
      .attr('refY', 0)
      .attr('markerWidth', 6)
      .attr('markerHeight', 6)
      .attr('orient', 'auto')
      .append('path')
      .attr('d', 'M0,-5L10,0L0,5')
      .attr('fill', '#ff6b6b');
    
    // Highlighted connection arrow marker  
    defs.append('marker')
      .attr('id', 'arrow-highlight')
      .attr('viewBox', '0 -5 10 10')
      .attr('refX', 8)
      .attr('refY', 0)
      .attr('markerWidth', 6)
      .attr('markerHeight', 6)
      .attr('orient', 'auto')
      .append('path')
      .attr('d', 'M0,-5L10,0L0,5')
      .attr('fill', '#ff6b6b');

    // Draw edges (connections) as directed arrows
    const linkSelection = container.selectAll('.link')
      .data(edges)
      .enter()
      .append('line')
      .classed('link', true)
      .attr('x1', d => xScale(nodes.find(n => n.id === d.from)?.x || 0))
      .attr('y1', d => yScale(nodes.find(n => n.id === d.from)?.y || 0))
      .attr('x2', d => {
        const fromNode = nodes.find(n => n.id === d.from);
        const toNode = nodes.find(n => n.id === d.to);
        if (!fromNode || !toNode) return 0;
        
        // Calculate shortened line to account for arrow head and node radius
        const dx = xScale(toNode.x) - xScale(fromNode.x);
        const dy = yScale(toNode.y) - yScale(fromNode.y);
        const length = Math.sqrt(dx * dx + dy * dy);
        const nodeRadius = 8; // Account for node size
        const shortenBy = nodeRadius + 2;
        
        return xScale(toNode.x) - (dx / length) * shortenBy;
      })
      .attr('y2', d => {
        const fromNode = nodes.find(n => n.id === d.from);
        const toNode = nodes.find(n => n.id === d.to);
        if (!fromNode || !toNode) return 0;
        
        // Calculate shortened line to account for arrow head and node radius
        const dx = xScale(toNode.x) - xScale(fromNode.x);
        const dy = yScale(toNode.y) - yScale(fromNode.y);
        const length = Math.sqrt(dx * dx + dy * dy);
        const nodeRadius = 8; // Account for node size
        const shortenBy = nodeRadius + 2;
        
        return yScale(toNode.y) - (dy / length) * shortenBy;
      })
      .attr('stroke', '#e0e0e0')
      .attr('stroke-width', 1)
      .attr('marker-end', 'url(#arrow)');

    // Create node groups
    const nodeSelection = container.selectAll('.node')
      .data(nodes)
      .enter()
      .append('g')
      .classed('node', true)
      .attr('transform', d => `translate(${xScale(d.x)}, ${yScale(d.y)})`)
      .style('cursor', 'pointer')
      .on('click', (event, d) => {
        setSelectedNode(d);
        
        // Highlight only outgoing connections (actual peers this node chose)
        const connectedNodes = new Set();
        edges.forEach(edge => {
          if (edge.from === d.id) {
            connectedNodes.add(edge.to);
          }
        });
        connectedNodes.add(d.id);
        setHighlightedNodes(connectedNodes);
      })
      .on('mouseover', function(event, d) {
        d3.select(this).select('circle').attr('r', 8);
        
        // Show tooltip
        const tooltip = d3.select('body').append('div')
          .attr('class', 'tooltip')
          .style('opacity', 0);
          
        tooltip.transition().duration(200).style('opacity', .9);
        tooltip.html(`Node ${d.id}<br/>Address: ${d.addr}<br/>Click to highlight connections`)
          .style('left', (event.pageX + 10) + 'px')
          .style('top', (event.pageY - 28) + 'px');
      })
      .on('mouseout', function(event, d) {
        d3.select(this).select('circle').attr('r', 6);
        d3.selectAll('.tooltip').remove();
      });

    // Add circles for nodes
    nodeSelection.append('circle')
      .attr('r', d => nodeStates.has(d.id) ? 8 : 6)
      .attr('fill', d => {
        const nodeState = nodeStates.get(d.id);
        if (selectedNode && d.id === selectedNode.id) return '#ff4757';
        if (highlightedNodes.has(d.id)) return '#ffa726';
        if (nodeState?.isOriginalSender) return '#e74c3c'; // Red for original sender
        if (nodeState?.hasMessage) return '#27ae60'; // Green for nodes with message
        
        // Color by cluster if clusters exist and no message state
        if (data.topology.clusters && data.topology.clusters.length > 0) {
          const nodeCluster = data.topology.clusters.find(cluster => 
            cluster.nodeIds.includes(d.id)
          );
          if (nodeCluster) {
            if (nodeCluster.isIsolated) {
              return highlightIsolated ? '#ff1744' : '#f44336'; // Brighter red when highlighting
            }
            if (highlightIsolated) {
              return '#bdbdbd'; // Gray out non-isolated clusters when highlighting
            }
            return clusterColors[nodeCluster.id % clusterColors.length];
          }
        }
        
        return '#4ecdc4'; // Default blue
      })
      .attr('stroke', '#fff')
      .attr('stroke-width', 2)
      .classed('message-node', d => nodeStates.has(d.id));

    // Add labels for nodes
    nodeSelection.append('text')
      .attr('dx', 10)
      .attr('dy', 4)
      .style('font-size', '10px')
      .style('fill', '#333')
      .style('pointer-events', 'none')
      .text(d => d.id);

    // Update link colors and markers based on highlighted state and message flow
    linkSelection.attr('stroke', d => {
      if (currentTrace && d.from === currentTrace.immediateForwarder && d.to === currentTrace.receiver) {
        return currentTrace.isDirect ? '#ff6b6b' : '#4ecdc4';
      }
      if (highlightedNodes.has(d.from) && highlightedNodes.has(d.to)) {
        return '#ff6b6b';
      }
      return '#e0e0e0';
    })
    .attr('stroke-width', d => {
      if (currentTrace && d.from === currentTrace.immediateForwarder && d.to === currentTrace.receiver) {
        return 3;
      }
      if (highlightedNodes.has(d.from) && highlightedNodes.has(d.to)) {
        return 2;
      }
      return 1;
    })
    .attr('marker-end', d => {
      if (currentTrace && d.from === currentTrace.immediateForwarder && d.to === currentTrace.receiver) {
        return 'url(#arrow-active)';
      }
      if (highlightedNodes.has(d.from) && highlightedNodes.has(d.to)) {
        return 'url(#arrow-highlight)';
      }
      return 'url(#arrow)';
    });

  }, [data, selectedNode, highlightedNodes, currentTrace, nodeStates]);

  // Reset node states when replay progress goes to 0
  useEffect(() => {
    if (replayProgress === 0) {
      setNodeStates(new Map());
    }
  }, [replayProgress]);

  const clearSelection = () => {
    setSelectedNode(null);
    setHighlightedNodes(new Set());
  };

  return (
    <div className="network-visualization">
      <div className="visualization-header">
        <h3>🌐 Network Topology</h3>
        <div className="header-controls">
          {data?.topology?.clusters && data.topology.clusters.some(c => c.isIsolated) && (
            <button 
              onClick={() => setHighlightIsolated(!highlightIsolated)}
              className={`isolation-toggle ${highlightIsolated ? 'active' : ''}`}
              title="Highlight isolated clusters"
            >
              {highlightIsolated ? '🔴 Showing Isolation' : '🔍 Show Isolation'}
            </button>
          )}
          {selectedNode && (
            <div className="selected-node-info">
              <span>Selected: Node {selectedNode.id} ({selectedNode.addr})</span>
              <button onClick={clearSelection} className="clear-btn">Clear Selection</button>
            </div>
          )}
        </div>
      </div>
      
      <div className="svg-container">
        <svg ref={svgRef}></svg>
      </div>
      
      {/* Cluster Statistics & Partition Analysis */}
      {data?.topology?.clusters && data.topology.clusters.length > 0 && (() => {
        const isolatedClusters = data.topology.clusters.filter(c => c.isIsolated);
        const connectedClusters = data.topology.clusters.filter(c => !c.isIsolated);
        const largestConnectedSize = connectedClusters.length > 0 ? Math.max(...connectedClusters.map(c => c.size)) : 0;
        const totalNodes = data.topology.nodes.length;
        const nodesInLargestComponent = largestConnectedSize;
        const networkConnectivity = (nodesInLargestComponent / totalNodes) * 100;

        return (
          <div className="cluster-stats">
            <h4>🔗 Network Analysis</h4>
            <div className="cluster-summary">
              <span>Total clusters: {data.topology.clusters.length}</span>
              <span>Isolated clusters: {isolatedClusters.length}</span>
              <span>Largest connected component: {largestConnectedSize} nodes ({networkConnectivity.toFixed(1)}%)</span>
            </div>
            
            {isolatedClusters.length > 0 && (
              <div className="partition-analysis">
                <h5>⚠️ Network Partitions Detected</h5>
                <p>
                  This network has {isolatedClusters.length} isolated cluster{isolatedClusters.length !== 1 ? 's' : ''} 
                  containing {isolatedClusters.reduce((sum, c) => sum + c.size, 0)} nodes total.
                  Messages cannot propagate between isolated clusters.
                </p>
                {isolatedClusters.length <= 3 && (
                  <div className="isolated-details">
                    {isolatedClusters.map(cluster => (
                      <span key={cluster.id} className="isolated-cluster-info">
                        Cluster {cluster.id}: {cluster.size} node{cluster.size !== 1 ? 's' : ''}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            )}
            
            <div className="cluster-list">
              {data.topology.clusters.slice(0, 6).map(cluster => (
                <div key={cluster.id} className="cluster-item">
                  <div 
                    className="cluster-color-indicator" 
                    style={{
                      backgroundColor: cluster.isIsolated ? '#f44336' : clusterColors[cluster.id % clusterColors.length]
                    }}
                  ></div>
                  <span>Cluster {cluster.id}: {cluster.size} nodes {cluster.isIsolated ? '(ISOLATED)' : ''}</span>
                </div>
              ))}
              {data.topology.clusters.length > 6 && (
                <div className="cluster-item">
                  <span>... and {data.topology.clusters.length - 6} more clusters</span>
                </div>
              )}
            </div>
          </div>
        );
      })()}

      <div className="legend">
        <div className="legend-item">
          <div className="legend-color" style={{backgroundColor: '#4ecdc4'}}></div>
          <span>Normal Node</span>
        </div>
        <div className="legend-item">
          <div className="legend-color" style={{backgroundColor: '#27ae60'}}></div>
          <span>Has Message</span>
        </div>
        <div className="legend-item">
          <div className="legend-color" style={{backgroundColor: '#e74c3c'}}></div>
          <span>Original Sender</span>
        </div>
        <div className="legend-item">
          <div className="legend-color" style={{backgroundColor: '#ff4757'}}></div>
          <span>Selected Node</span>
        </div>
        <div className="legend-item">
          <div className="legend-color" style={{backgroundColor: '#ffa726'}}></div>
          <span>Connected Node</span>
        </div>
        <div className="legend-item">
          <div className="legend-color" style={{backgroundColor: '#f44336'}}></div>
          <span>Isolated Cluster</span>
        </div>
      </div>
      
      <div className="controls">
        <p>🖱️ Click nodes to highlight connections • Use mouse wheel to zoom • Drag to pan</p>
      </div>
    </div>
  );
};

export default NetworkVisualization;