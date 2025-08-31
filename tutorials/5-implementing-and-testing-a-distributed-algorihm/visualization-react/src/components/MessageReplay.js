import React, { useState, useEffect, useCallback, useRef } from 'react';
import './MessageReplay.css';

const MessageReplay = ({ traces, onTraceChange, onProgressChange, onPlayStateChange }) => {
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentIndex, setCurrentIndex] = useState(0);
  const [playbackSpeed, setPlaybackSpeed] = useState(1);
  const [progress, setProgress] = useState(0);
  const [showDetails, setShowDetails] = useState(true);
  const intervalRef = useRef(null);

  // Sort traces by timestamp for chronological playback
  const sortedTraces = traces.sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp));

  const totalDuration = sortedTraces.length > 0 ? 
    new Date(sortedTraces[sortedTraces.length - 1].timestamp) - new Date(sortedTraces[0].timestamp) : 0;

  const play = useCallback(() => {
    if (currentIndex >= sortedTraces.length - 1) {
      setCurrentIndex(0);
    }
    setIsPlaying(true);
    onPlayStateChange?.(true);
  }, [currentIndex, sortedTraces.length, onPlayStateChange]);

  const pause = useCallback(() => {
    setIsPlaying(false);
    onPlayStateChange?.(false);
  }, [onPlayStateChange]);

  const reset = useCallback(() => {
    setIsPlaying(false);
    onPlayStateChange?.(false);
    setCurrentIndex(0);
    setProgress(0);
  }, [onPlayStateChange]);

  const stepForward = useCallback(() => {
    if (currentIndex < sortedTraces.length - 1) {
      setCurrentIndex(prev => prev + 1);
    }
  }, [currentIndex, sortedTraces.length]);

  const stepBackward = useCallback(() => {
    if (currentIndex > 0) {
      setCurrentIndex(prev => prev - 1);
    }
  }, [currentIndex]);

  const handleProgressChange = useCallback((event) => {
    const newProgress = parseFloat(event.target.value);
    const newIndex = Math.floor((newProgress / 100) * (sortedTraces.length - 1));
    setCurrentIndex(newIndex);
    setProgress(newProgress);
  }, [sortedTraces.length]);

  useEffect(() => {
    if (isPlaying && currentIndex < sortedTraces.length - 1) {
      intervalRef.current = setInterval(() => {
        setCurrentIndex(prev => {
          if (prev >= sortedTraces.length - 1) {
            setIsPlaying(false);
            return prev;
          }
          return prev + 1;
        });
      }, 100 / playbackSpeed); // Base speed: 100ms per step
    } else {
      clearInterval(intervalRef.current);
    }

    return () => clearInterval(intervalRef.current);
  }, [isPlaying, currentIndex, playbackSpeed, sortedTraces.length]);

  useEffect(() => {
    const newProgress = (currentIndex / (sortedTraces.length - 1)) * 100;
    const progressValue = isNaN(newProgress) ? 0 : newProgress;
    setProgress(progressValue);
    onProgressChange?.(progressValue);
    
    // Update current trace
    if (sortedTraces[currentIndex]) {
      onTraceChange?.(sortedTraces[currentIndex]);
    }
  }, [currentIndex, sortedTraces.length, onProgressChange, onTraceChange, sortedTraces]);

  if (!traces || traces.length === 0) {
    return (
      <div className="message-replay">
        <h3>📡 Message Replay</h3>
        <p>No message traces available for replay.</p>
      </div>
    );
  }

  const currentTrace = sortedTraces[currentIndex];
  const currentTime = currentTrace ? new Date(currentTrace.timestamp) : null;
  const startTime = sortedTraces[0] ? new Date(sortedTraces[0].timestamp) : null;
  const elapsedTime = currentTime && startTime ? currentTime - startTime : 0;

  // Get recent traces for the activity feed (last 5)
  const recentTraces = sortedTraces.slice(Math.max(0, currentIndex - 4), currentIndex + 1).reverse();

  // Calculate statistics for current point in time
  const tracesUpToNow = sortedTraces.slice(0, currentIndex + 1);
  const uniqueNodes = new Set();
  const messageIds = new Set();
  tracesUpToNow.forEach(trace => {
    uniqueNodes.add(trace.receiver);
    messageIds.add(trace.messageId);
  });

  return (
    <div className="message-replay">
      <div className="replay-header">
        <h3>📡 Message Replay</h3>
        <div className="replay-stats">
          <span className="stat-item">
            📊 {currentIndex + 1} / {sortedTraces.length} messages
          </span>
          <span className="stat-item">
            🎯 {uniqueNodes.size} nodes reached
          </span>
          <span className="stat-item">
            ⏱️ +{Math.round(elapsedTime / 1000)}s
          </span>
        </div>
      </div>

      <div className="replay-controls">
        <div className="control-buttons">
          <button onClick={reset} disabled={currentIndex === 0} title="Reset">
            ⏮️
          </button>
          <button onClick={stepBackward} disabled={currentIndex === 0} title="Step Back">
            ⏪
          </button>
          <button 
            onClick={isPlaying ? pause : play} 
            className="play-pause-btn"
            title={isPlaying ? "Pause" : "Play"}
          >
            {isPlaying ? '⏸️' : '▶️'}
          </button>
          <button 
            onClick={stepForward} 
            disabled={currentIndex >= sortedTraces.length - 1}
            title="Step Forward"
          >
            ⏩
          </button>
        </div>

        <div className="progress-control">
          <input
            type="range"
            min="0"
            max="100"
            value={progress}
            onChange={handleProgressChange}
            className="progress-slider"
          />
          <div className="progress-labels">
            <span>Start</span>
            <span>{Math.round(progress)}%</span>
            <span>End</span>
          </div>
        </div>

        <div className="speed-control">
          <label>Speed:</label>
          <select 
            value={playbackSpeed} 
            onChange={(e) => setPlaybackSpeed(parseFloat(e.target.value))}
          >
            <option value={0.25}>0.25x</option>
            <option value={0.5}>0.5x</option>
            <option value={1}>1x</option>
            <option value={2}>2x</option>
            <option value={4}>4x</option>
            <option value={8}>8x</option>
          </select>
        </div>
      </div>

      {currentTrace && (
        <div className="current-message">
          <div className="message-header">
            <h4>Current Message</h4>
            <button 
              onClick={() => setShowDetails(!showDetails)}
              className="toggle-details-btn"
            >
              {showDetails ? '➖' : '➕'} Details
            </button>
          </div>
          
          <div className="message-flow">
            <div className="flow-item sender">
              <span className="flow-label">Original Sender</span>
              <span className="flow-value">Node {currentTrace.originalSender}</span>
            </div>
            
            {!currentTrace.isDirect && (
              <>
                <div className="flow-arrow">→</div>
                <div className="flow-item forwarder">
                  <span className="flow-label">Forwarder</span>
                  <span className="flow-value">Node {currentTrace.immediateForwarder}</span>
                </div>
              </>
            )}
            
            <div className="flow-arrow">→</div>
            <div className="flow-item receiver">
              <span className="flow-label">Receiver</span>
              <span className="flow-value">Node {currentTrace.receiver}</span>
            </div>
          </div>

          {showDetails && (
            <div className="message-details">
              <div className="detail-grid">
                <div className="detail-item">
                  <span className="detail-label">Message ID:</span>
                  <span className="detail-value">{currentTrace.messageId.substring(0, 8)}...</span>
                </div>
                <div className="detail-item">
                  <span className="detail-label">Content:</span>
                  <span className="detail-value">"{currentTrace.content}"</span>
                </div>
                <div className="detail-item">
                  <span className="detail-label">TTL:</span>
                  <span className="detail-value">{currentTrace.ttl}</span>
                </div>
                <div className="detail-item">
                  <span className="detail-label">Transmission:</span>
                  <span className={`detail-value ${currentTrace.isDirect ? 'direct' : 'forwarded'}`}>
                    {currentTrace.isDirect ? '🎯 Direct' : '🔄 Forwarded'}
                  </span>
                </div>
                <div className="detail-item">
                  <span className="detail-label">Timestamp:</span>
                  <span className="detail-value">
                    {new Date(currentTrace.timestamp).toLocaleTimeString()}
                  </span>
                </div>
              </div>
            </div>
          )}
        </div>
      )}

      <div className="activity-feed">
        <h4>📋 Recent Activity</h4>
        <div className="activity-list">
          {recentTraces.map((trace, index) => (
            <div 
              key={`${trace.messageId}-${trace.receiver}`}
              className={`activity-item ${index === 0 ? 'current' : ''}`}
            >
              <div className="activity-main">
                <span className="activity-text">
                  Node {trace.receiver} received from Node {trace.originalSender}
                  {!trace.isDirect && ` (via Node ${trace.immediateForwarder})`}
                </span>
                <span className="activity-time">
                  {new Date(trace.timestamp).toLocaleTimeString()}
                </span>
              </div>
              <div className="activity-meta">
                <span className={`transmission-type ${trace.isDirect ? 'direct' : 'forwarded'}`}>
                  {trace.isDirect ? 'Direct' : 'Forwarded'}
                </span>
                <span className="ttl">TTL: {trace.ttl}</span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export default MessageReplay;