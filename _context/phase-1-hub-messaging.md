# Phase 1 Implementation Guide: Hub and Messaging Foundation

## Overview

This guide provides step-by-step instructions for implementing Phase 1 of go-agents-orchestration: the messaging and hub coordination primitives.

**Goal**: Port and refine the working hub implementation from go-agents-research, replacing research dependencies with production-ready equivalents.

**Scope**:
- `messaging/` package: Message structures and builders
- `hub/` package: Agent coordination and routing
- Multi-hub coordination support
- Integration with go-agents

**Not in Scope** (Future Phases):
- State graph execution (Phase 2)
- Workflow patterns (Phase 3)
- Advanced observability (Phase 4)

## Implementation Strategy

### Port from Research

**Source**: `github.com/JaimeStill/go-agents-research/hub/`

**Approach**:
1. Copy working code from research
2. Replace research dependencies:
   - `config.HubConfig` → `github.com/JaimeStill/go-agents-orchestration/config`
   - `errors.*` → Standard Go errors (refine later)
   - `metrics.HubMetrics` → Minimal metrics in `hub/`
   - `messaging.AgentMessage` → `github.com/JaimeStill/go-agents-orchestration/messaging`
3. Simplify message structure (essential fields only)
4. Follow go-agents conventions

### Bottom-Up Construction

**Order**:
1. `messaging/` (Level 1 - no dependencies)
2. `hub/` (Level 2 - depends on messaging)
3. Tests for both packages
4. Integration examples

## Step 1: Create messaging Package

### messaging/types.go

Define message types and priority levels:

```go
package messaging

// MessageType defines the type of inter-agent message.
type MessageType string

const (
	MessageTypeRequest      MessageType = "request"
	MessageTypeResponse     MessageType = "response"
	MessageTypeNotification MessageType = "notification"
	MessageTypeBroadcast    MessageType = "broadcast"
)

// Priority defines the priority level of a message.
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)
```

### messaging/message.go

Define the core message structure:

```go
package messaging

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"maps"
	"time"
)

// Message represents a message sent between agents with routing metadata.
type Message struct {
	// Core message fields
	ID   string      `json:"id"`
	From string      `json:"from"`
	To   string      `json:"to"`
	Type MessageType `json:"type"`
	Data any         `json:"data"`

	// Routing and correlation
	ReplyTo string `json:"reply_to,omitempty"`
	Topic   string `json:"topic,omitempty"`

	// Metadata
	Timestamp time.Time `json:"timestamp"`
	Priority  Priority  `json:"priority,omitempty"`

	// Additional metadata
	Headers map[string]string `json:"headers,omitempty"`
}

// IsRequest returns true if this is a request message.
func (msg *Message) IsRequest() bool {
	return msg.Type == MessageTypeRequest
}

// IsResponse returns true if this is a response message.
func (msg *Message) IsResponse() bool {
	return msg.Type == MessageTypeResponse
}

// IsBroadcast returns true if this is a broadcast message.
func (msg *Message) IsBroadcast() bool {
	return msg.Type == MessageTypeBroadcast
}

// Clone creates a deep copy of the message.
func (msg *Message) Clone() *Message {
	clone := *msg

	// Copy headers map
	if msg.Headers != nil {
		clone.Headers = make(map[string]string)
		maps.Copy(clone.Headers, msg.Headers)
	}

	return &clone
}

// String returns a string representation of the message for logging.
func (msg *Message) String() string {
	return fmt.Sprintf("Message{ID: %s, From: %s, To: %s, Type: %s, Topic: %s}",
		msg.ID, msg.From, msg.To, msg.Type, msg.Topic)
}

// generateMessageID generates a unique message ID using crypto/rand.
func generateMessageID() string {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("msg_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}
```

### messaging/builder.go

Implement fluent message builders:

```go
package messaging

import "time"

// MessageBuilder provides a fluent API for constructing messages.
type MessageBuilder struct {
	message *Message
}

// NewMessage creates a new message builder with required fields.
func NewMessage(from, to string, messageType MessageType, data any) *MessageBuilder {
	return &MessageBuilder{
		message: &Message{
			ID:        generateMessageID(),
			From:      from,
			To:        to,
			Type:      messageType,
			Data:      data,
			Timestamp: time.Now(),
			Priority:  PriorityNormal,
		},
	}
}

// NewRequest creates a new request message builder.
func NewRequest(from, to string, data any) *MessageBuilder {
	return NewMessage(from, to, MessageTypeRequest, data)
}

// NewResponse creates a new response message builder for replying to a request.
func NewResponse(from, to, replyTo string, data any) *MessageBuilder {
	return NewMessage(from, to, MessageTypeResponse, data).ReplyTo(replyTo)
}

// NewNotification creates a new notification message builder.
func NewNotification(from, to string, data any) *MessageBuilder {
	return NewMessage(from, to, MessageTypeNotification, data)
}

// NewBroadcast creates a new broadcast message builder.
func NewBroadcast(from string, data any) *MessageBuilder {
	return NewMessage(from, "*", MessageTypeBroadcast, data)
}

// ReplyTo sets the reply-to field for response correlation.
func (mb *MessageBuilder) ReplyTo(replyTo string) *MessageBuilder {
	mb.message.ReplyTo = replyTo
	return mb
}

// Topic sets the topic for pub/sub messaging.
func (mb *MessageBuilder) Topic(topic string) *MessageBuilder {
	mb.message.Topic = topic
	return mb
}

// Priority sets the message priority.
func (mb *MessageBuilder) Priority(priority Priority) *MessageBuilder {
	mb.message.Priority = priority
	return mb
}

// Headers sets custom headers.
func (mb *MessageBuilder) Headers(headers map[string]string) *MessageBuilder {
	mb.message.Headers = headers
	return mb
}

// Build returns the constructed message.
func (mb *MessageBuilder) Build() *Message {
	return mb.message
}
```

### messaging Package Documentation

Add `messaging/doc.go`:

```go
// Package messaging provides message structures and builders for inter-agent communication.
//
// The messaging package defines the core message types used throughout the orchestration
// system for agent-to-agent communication. Messages are immutable once built and support
// various communication patterns including request/response, notifications, and broadcasts.
//
// # Message Types
//
// - Request: Synchronous request expecting a response
// - Response: Reply to a request with correlation via ReplyTo field
// - Notification: Asynchronous fire-and-forget message
// - Broadcast: Message sent to all agents except sender
//
// # Usage
//
//	// Create a request
//	msg := messaging.NewRequest("agent-a", "agent-b", taskData).
//	    Priority(messaging.PriorityHigh).
//	    Build()
//
//	// Create a response
//	resp := messaging.NewResponse("agent-b", "agent-a", msg.ID, resultData).Build()
//
//	// Create a notification
//	notif := messaging.NewNotification("agent-a", "agent-b", statusUpdate).Build()
package messaging
```

## Step 2: Create hub Package

### hub/agent.go

Define the minimal agent interface:

```go
package hub

// Agent represents the minimal interface needed for hub registration.
type Agent interface {
	ID() string
}
```

### hub/handler.go

Define message handler types:

```go
package hub

import (
	"context"

	"github.com/JaimeStill/go-agents-orchestration/messaging"
)

// MessageContext provides context information to message handlers.
type MessageContext struct {
	HubName string
	Agent   Agent
}

// MessageHandler processes incoming messages for an agent.
type MessageHandler func(ctx context.Context, message *messaging.Message, context *MessageContext) (*messaging.Message, error)
```

### hub/channel.go

Port the message channel implementation from research:

```go
package hub

import (
	"context"
	"sync/atomic"
	"time"
)

// MessageChannel provides a context-aware channel wrapper for type-safe messaging.
type MessageChannel[T any] struct {
	channel    chan T
	context    context.Context
	bufferSize int
	closed     int32
}

// NewMessageChannel creates a new message channel.
func NewMessageChannel[T any](ctx context.Context, bufferSize int) *MessageChannel[T] {
	return &MessageChannel[T]{
		channel:    make(chan T, bufferSize),
		context:    ctx,
		bufferSize: bufferSize,
	}
}

// Send sends a message through the channel with context awareness.
func (mc *MessageChannel[T]) Send(ctx context.Context, message T) error {
	select {
	case mc.channel <- message:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-mc.context.Done():
		return mc.context.Err()
	}
}

// Receive receives a message from the channel with context awareness.
func (mc *MessageChannel[T]) Receive(ctx context.Context) (T, error) {
	select {
	case message := <-mc.channel:
		return message, nil
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	case <-mc.context.Done():
		var zero T
		return zero, mc.context.Err()
	}
}

// TryReceive attempts to receive a message without blocking.
func (mc *MessageChannel[T]) TryReceive() (T, bool) {
	select {
	case message := <-mc.channel:
		return message, true
	default:
		var zero T
		return zero, false
	}
}

// Close closes the channel.
func (mc *MessageChannel[T]) Close() {
	if atomic.CompareAndSwapInt32(&mc.closed, 0, 1) {
		close(mc.channel)
	}
}

// IsClosed returns whether the channel is closed.
func (mc *MessageChannel[T]) IsClosed() bool {
	return atomic.LoadInt32(&mc.closed) == 1
}

// BufferSize returns the channel buffer size.
func (mc *MessageChannel[T]) BufferSize() int {
	return mc.bufferSize
}

// QueueLength returns the current number of messages in the channel.
func (mc *MessageChannel[T]) QueueLength() int {
	return len(mc.channel)
}
```

### hub/metrics.go

Create minimal metrics structure:

```go
package hub

import "sync/atomic"

// Metrics provides hub performance metrics.
type Metrics struct {
	localAgents  atomic.Int64
	messagesSent atomic.Int64
	messagesRecv atomic.Int64
}

// NewMetrics creates a new metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{}
}

// RecordLocalAgent adjusts the local agent count.
func (m *Metrics) RecordLocalAgent(delta int) {
	m.localAgents.Add(int64(delta))
}

// RecordMessage records a sent or received message.
func (m *Metrics) RecordMessage(delta int) {
	m.messagesSent.Add(int64(delta))
}

// Snapshot returns current metric values.
type MetricsSnapshot struct {
	LocalAgents  int64
	MessagesSent int64
	MessagesRecv int64
}

// GetSnapshot returns a snapshot of current metrics.
func (m *Metrics) GetSnapshot() MetricsSnapshot {
	return MetricsSnapshot{
		LocalAgents:  m.localAgents.Load(),
		MessagesSent: m.messagesSent.Load(),
		MessagesRecv: m.messagesRecv.Load(),
	}
}
```

### hub/registry.go

Define agent registration structure:

```go
package hub

import (
	"time"

	"github.com/JaimeStill/go-agents-orchestration/messaging"
)

// registration represents an agent registered in the hub.
type registration struct {
	Agent    Agent
	Handler  MessageHandler
	Channel  *MessageChannel[*messaging.Message]
	LastSeen time.Time
}
```

### hub/hub.go

Port the hub implementation from research:

```go
package hub

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/JaimeStill/go-agents-orchestration/config"
	"github.com/JaimeStill/go-agents-orchestration/messaging"
)

// Hub provides agent coordination and message routing.
type Hub interface {
	// Agent lifecycle
	RegisterAgent(agent Agent, handler MessageHandler) error
	UnregisterAgent(agentID string) error

	// Core messaging
	Send(ctx context.Context, from, to string, data any) error
	Request(ctx context.Context, from, to string, data any) (*messaging.Message, error)
	Broadcast(ctx context.Context, from string, data any) error

	// Pub/Sub
	Subscribe(agentID, topic string) error
	Publish(ctx context.Context, from, topic string, data any) error

	// Management
	GetMetrics() MetricsSnapshot
	Shutdown(timeout time.Duration) error
}

// hub implements Hub with agent coordination and message routing.
type hub struct {
	name string

	// Agent registry
	agents      map[string]*registration
	agentsMutex sync.RWMutex

	// Request-response correlation
	responseChannels map[string]chan *messaging.Message
	responsesMutex   sync.RWMutex

	// Pub/Sub
	subscriptions map[string]map[string]*registration // topic -> agentID -> agent
	subsMutex     sync.RWMutex

	// Configuration
	channelBufferSize int
	defaultTimeout    time.Duration

	// Observability
	logger  *slog.Logger
	metrics *Metrics

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// New creates a new Hub instance.
func New(ctx context.Context, hubConfig config.HubConfig) Hub {
	hubCtx, cancel := context.WithCancel(ctx)

	h := &hub{
		name:              hubConfig.Name,
		agents:            make(map[string]*registration),
		responseChannels:  make(map[string]chan *messaging.Message),
		subscriptions:     make(map[string]map[string]*registration),
		channelBufferSize: hubConfig.ChannelBufferSize,
		defaultTimeout:    hubConfig.DefaultTimeout,
		logger:            hubConfig.Logger,
		metrics:           NewMetrics(),
		ctx:               hubCtx,
		cancel:            cancel,
		done:              make(chan struct{}),
	}

	// Start background message processing
	go h.messageLoop()

	return h
}

// RegisterAgent registers an agent with its message handler.
func (h *hub) RegisterAgent(agent Agent, handler MessageHandler) error {
	agentID := agent.ID()

	h.agentsMutex.Lock()
	defer h.agentsMutex.Unlock()

	if _, exists := h.agents[agentID]; exists {
		return fmt.Errorf("agent already registered: %s", agentID)
	}

	channel := NewMessageChannel[*messaging.Message](h.ctx, h.channelBufferSize)

	reg := &registration{
		Agent:    agent,
		Handler:  handler,
		Channel:  channel,
		LastSeen: time.Now(),
	}

	h.agents[agentID] = reg
	h.metrics.RecordLocalAgent(1)

	h.logger.DebugContext(h.ctx, "agent registered",
		slog.String("hub_name", h.name),
		slog.String("agent_id", agentID))

	return nil
}

// UnregisterAgent removes an agent from the hub.
func (h *hub) UnregisterAgent(agentID string) error {
	h.agentsMutex.Lock()
	reg, exists := h.agents[agentID]
	if exists {
		delete(h.agents, agentID)
		reg.Channel.Close()
	}
	h.agentsMutex.Unlock()

	if !exists {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	// Clean up subscriptions
	h.subsMutex.Lock()
	for topic, subs := range h.subscriptions {
		if _, exists := subs[agentID]; exists {
			delete(subs, agentID)
			if len(subs) == 0 {
				delete(h.subscriptions, topic)
			}
		}
	}
	h.subsMutex.Unlock()

	h.metrics.RecordLocalAgent(-1)
	h.logger.DebugContext(h.ctx, "agent unregistered",
		slog.String("hub_name", h.name),
		slog.String("agent_id", agentID))

	return nil
}

// Send sends a message to another agent (fire-and-forget).
func (h *hub) Send(ctx context.Context, from, to string, data any) error {
	h.agentsMutex.RLock()
	reg, exists := h.agents[to]
	h.agentsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("destination agent not found: %s", to)
	}

	message := messaging.NewMessage(from, to, messaging.MessageTypeNotification, data).Build()
	err := reg.Channel.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to deliver message: %w", err)
	}

	h.updateLastSeen(from)
	h.metrics.RecordMessage(1)

	return nil
}

// Request sends a request and waits for a response with proper correlation.
func (h *hub) Request(ctx context.Context, from, to string, data any) (*messaging.Message, error) {
	h.agentsMutex.RLock()
	reg, exists := h.agents[to]
	h.agentsMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("destination agent not found: %s", to)
	}

	message := messaging.NewRequest(from, to, data).Build()
	responseChannel := make(chan *messaging.Message, 1)

	// Register response channel for correlation
	h.responsesMutex.Lock()
	h.responseChannels[message.ID] = responseChannel
	h.responsesMutex.Unlock()

	// Clean up response channel when done
	defer func() {
		h.responsesMutex.Lock()
		delete(h.responseChannels, message.ID)
		h.responsesMutex.Unlock()
		close(responseChannel)
	}()

	// Send the request
	err := reg.Channel.Send(ctx, message)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	h.updateLastSeen(from)

	// Wait for response with timeout
	timeout := h.defaultTimeout
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	}

	select {
	case response := <-responseChannel:
		return response, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("request cancelled: %w", ctx.Err())
	case <-time.After(timeout):
		return nil, fmt.Errorf("request timeout after %v", timeout)
	}
}

// Broadcast sends a message to all registered agents except the sender.
func (h *hub) Broadcast(ctx context.Context, from string, data any) error {
	h.agentsMutex.RLock()
	registrations := make([]*registration, 0, len(h.agents))
	for agentID, reg := range h.agents {
		if agentID != from { // Skip sender
			registrations = append(registrations, reg)
		}
	}
	h.agentsMutex.RUnlock()

	// Send to all agents
	delivered := 0
	for _, reg := range registrations {
		message := messaging.NewMessage(from, reg.Agent.ID(), messaging.MessageTypeBroadcast, data).Build()
		if err := reg.Channel.Send(ctx, message); err != nil {
			h.logger.WarnContext(ctx, "failed to deliver broadcast",
				slog.String("hub_name", h.name),
				slog.String("from", from),
				slog.String("to", reg.Agent.ID()),
				slog.String("error", err.Error()))
		} else {
			delivered++
		}
	}

	h.updateLastSeen(from)
	h.logger.DebugContext(ctx, "broadcast sent",
		slog.String("hub_name", h.name),
		slog.String("from", from),
		slog.Int("recipients", len(registrations)),
		slog.Int("delivered", delivered))

	return nil
}

// Subscribe subscribes an agent to a topic for pub/sub messaging.
func (h *hub) Subscribe(agentID, topic string) error {
	h.agentsMutex.RLock()
	reg, exists := h.agents[agentID]
	h.agentsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	h.subsMutex.Lock()
	if h.subscriptions[topic] == nil {
		h.subscriptions[topic] = make(map[string]*registration)
	}
	h.subscriptions[topic][agentID] = reg
	h.subsMutex.Unlock()

	h.logger.DebugContext(h.ctx, "agent subscribed to topic",
		slog.String("hub_name", h.name),
		slog.String("agent_id", agentID),
		slog.String("topic", topic))

	return nil
}

// Publish publishes a message to all subscribers of a topic.
func (h *hub) Publish(ctx context.Context, from, topic string, data any) error {
	h.subsMutex.RLock()
	subscribers, exists := h.subscriptions[topic]
	if !exists {
		h.subsMutex.RUnlock()
		h.logger.DebugContext(ctx, "no subscribers for topic",
			slog.String("hub_name", h.name),
			slog.String("topic", topic))
		return nil
	}

	// Copy subscribers to avoid holding lock during delivery
	subscriberList := make([]*registration, 0, len(subscribers))
	for _, reg := range subscribers {
		subscriberList = append(subscriberList, reg)
	}
	h.subsMutex.RUnlock()

	// Deliver to all subscribers
	delivered := 0
	for _, reg := range subscriberList {
		if reg.Agent.ID() == from {
			continue // Skip self-delivery
		}

		message := messaging.NewMessage(from, reg.Agent.ID(), messaging.MessageTypeNotification, data).Topic(topic).Build()
		if err := reg.Channel.Send(ctx, message); err != nil {
			h.logger.WarnContext(ctx, "failed to deliver published message",
				slog.String("hub_name", h.name),
				slog.String("topic", topic),
				slog.String("subscriber", reg.Agent.ID()),
				slog.String("error", err.Error()))
		} else {
			delivered++
		}
	}

	h.updateLastSeen(from)
	h.logger.DebugContext(ctx, "message published",
		slog.String("hub_name", h.name),
		slog.String("topic", topic),
		slog.Int("subscribers", len(subscriberList)),
		slog.Int("delivered", delivered))

	return nil
}

// GetMetrics returns hub metrics.
func (h *hub) GetMetrics() MetricsSnapshot {
	return h.metrics.GetSnapshot()
}

// Shutdown gracefully shuts down the hub.
func (h *hub) Shutdown(timeout time.Duration) error {
	h.logger.DebugContext(h.ctx, "shutting down hub",
		slog.String("hub_name", h.name))
	h.cancel()

	select {
	case <-h.done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("hub shutdown timeout after %v", timeout)
	}
}

// messageLoop processes incoming messages for all agents.
func (h *hub) messageLoop() {
	defer close(h.done)

	for {
		select {
		case <-h.ctx.Done():
			return
		default:
			h.processAgentMessages()
		}
	}
}

// processAgentMessages efficiently processes messages from all registered agents.
func (h *hub) processAgentMessages() {
	h.agentsMutex.RLock()
	if len(h.agents) == 0 {
		h.agentsMutex.RUnlock()
		return
	}

	registrations := make([]*registration, 0, len(h.agents))
	for _, reg := range h.agents {
		registrations = append(registrations, reg)
	}
	h.agentsMutex.RUnlock()

	// Check each agent channel for messages
	for _, reg := range registrations {
		select {
		case <-h.ctx.Done():
			return
		default:
			// Non-blocking check for messages
			if message, ok := reg.Channel.TryReceive(); ok && message != nil {
				go h.handleMessage(reg, message)
			}
		}
	}
}

// handleMessage processes a message for an agent.
func (h *hub) handleMessage(reg *registration, message *messaging.Message) {
	if reg.Handler == nil {
		return
	}

	context := &MessageContext{
		HubName: h.name,
		Agent:   reg.Agent,
	}

	response, err := reg.Handler(h.ctx, message, context)
	if err != nil {
		h.logger.ErrorContext(h.ctx, "message handler failed",
			slog.String("hub_name", h.name),
			slog.String("agent_id", reg.Agent.ID()),
			slog.String("from", message.From),
			slog.String("error", err.Error()))
		return
	}

	// If there's a response, handle it appropriately
	if response != nil {
		// Check if this is a response to a pending request
		if response.Type == messaging.MessageTypeResponse && response.ReplyTo != "" {
			h.responsesMutex.RLock()
			respChan, exists := h.responseChannels[response.ReplyTo]
			h.responsesMutex.RUnlock()

			if exists {
				// Send to response channel for request correlation
				select {
				case respChan <- response:
					// Response delivered to waiting request
				default:
					// Response channel full or closed
				}
				return
			}
		}

		// Regular message routing
		h.agentsMutex.RLock()
		targetReg, exists := h.agents[response.To]
		h.agentsMutex.RUnlock()

		if exists {
			if err := targetReg.Channel.Send(h.ctx, response); err != nil {
				h.logger.ErrorContext(h.ctx, "failed to send response",
					slog.String("hub_name", h.name),
					slog.String("from", response.From),
					slog.String("to", response.To),
					slog.String("error", err.Error()))
			}
		}
	}
}

// updateLastSeen updates the last seen timestamp for an agent.
func (h *hub) updateLastSeen(agentID string) {
	h.agentsMutex.Lock()
	if reg, exists := h.agents[agentID]; exists {
		reg.LastSeen = time.Now()
	}
	h.agentsMutex.Unlock()
}
```

### hub Package Documentation

Add `hub/doc.go`:

```go
// Package hub provides agent coordination and message routing primitives.
//
// The hub package implements a coordination primitive that allows agents to
// register, send messages, and communicate through various patterns including
// request/response, broadcast, and pub/sub.
//
// # Core Concepts
//
// Hub: Persistent networking fabric for agent coordination
// Agent: Minimal interface (ID()) required for hub registration
// MessageHandler: Callback function that processes incoming messages
//
// # Communication Patterns
//
// - Send: Fire-and-forget message delivery
// - Request: Synchronous request with response correlation
// - Broadcast: Send to all agents except sender
// - Pub/Sub: Topic-based subscription messaging
//
// # Multi-Hub Coordination
//
// Agents can register with multiple hubs simultaneously, enabling fractal
// growth patterns where hubs network through shared agents.
//
// # Usage
//
//	// Create hub
//	hubConfig := config.DefaultHubConfig()
//	h := hub.New(ctx, hubConfig)
//
//	// Register agent
//	handler := func(ctx context.Context, msg *messaging.Message, msgCtx *hub.MessageContext) (*messaging.Message, error) {
//	    // Process message
//	    return response, nil
//	}
//	h.RegisterAgent(agent, handler)
//
//	// Send message
//	h.Send(ctx, "agent-a", "agent-b", data)
package hub
```

## Step 3: Testing

### tests/messaging/message_test.go

Test message creation and helpers:

```go
package messaging_test

import (
	"testing"

	"github.com/JaimeStill/go-agents-orchestration/messaging"
)

func TestMessage_Builders(t *testing.T) {
	tests := []struct {
		name     string
		builder  func() *messaging.Message
		wantType messaging.MessageType
		wantFrom string
		wantTo   string
	}{
		{
			name: "NewRequest",
			builder: func() *messaging.Message {
				return messaging.NewRequest("agent-a", "agent-b", "test").Build()
			},
			wantType: messaging.MessageTypeRequest,
			wantFrom: "agent-a",
			wantTo:   "agent-b",
		},
		{
			name: "NewResponse",
			builder: func() *messaging.Message {
				return messaging.NewResponse("agent-b", "agent-a", "msg-123", "result").Build()
			},
			wantType: messaging.MessageTypeResponse,
			wantFrom: "agent-b",
			wantTo:   "agent-a",
		},
		{
			name: "NewNotification",
			builder: func() *messaging.Message {
				return messaging.NewNotification("agent-a", "agent-b", "update").Build()
			},
			wantType: messaging.MessageTypeNotification,
			wantFrom: "agent-a",
			wantTo:   "agent-b",
		},
		{
			name: "NewBroadcast",
			builder: func() *messaging.Message {
				return messaging.NewBroadcast("agent-a", "announcement").Build()
			},
			wantType: messaging.MessageTypeBroadcast,
			wantFrom: "agent-a",
			wantTo:   "*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.builder()

			if msg.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", msg.Type, tt.wantType)
			}
			if msg.From != tt.wantFrom {
				t.Errorf("From = %v, want %v", msg.From, tt.wantFrom)
			}
			if msg.To != tt.wantTo {
				t.Errorf("To = %v, want %v", msg.To, tt.wantTo)
			}
			if msg.ID == "" {
				t.Error("ID should not be empty")
			}
		})
	}
}
```

### tests/hub/hub_test.go

Test hub coordination patterns:

```go
package hub_test

import (
	"context"
	"testing"
	"time"

	"github.com/JaimeStill/go-agents-orchestration/config"
	"github.com/JaimeStill/go-agents-orchestration/hub"
	"github.com/JaimeStill/go-agents-orchestration/messaging"
)

// testAgent implements hub.Agent for testing
type testAgent struct {
	id string
}

func (a *testAgent) ID() string { return a.id }

func TestHub_RegisterAgent(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultHubConfig()
	h := hub.New(ctx, cfg)
	defer h.Shutdown(5 * time.Second)

	agent := &testAgent{id: "test-agent"}
	handler := func(ctx context.Context, msg *messaging.Message, msgCtx *hub.MessageContext) (*messaging.Message, error) {
		return nil, nil
	}

	err := h.RegisterAgent(agent, handler)
	if err != nil {
		t.Fatalf("RegisterAgent() error = %v", err)
	}

	// Duplicate registration should fail
	err = h.RegisterAgent(agent, handler)
	if err == nil {
		t.Error("RegisterAgent() should fail for duplicate registration")
	}
}

func TestHub_Send(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultHubConfig()
	h := hub.New(ctx, cfg)
	defer h.Shutdown(5 * time.Second)

	received := make(chan string, 1)

	agentA := &testAgent{id: "agent-a"}
	agentB := &testAgent{id: "agent-b"}

	handlerA := func(ctx context.Context, msg *messaging.Message, msgCtx *hub.MessageContext) (*messaging.Message, error) {
		return nil, nil
	}

	handlerB := func(ctx context.Context, msg *messaging.Message, msgCtx *hub.MessageContext) (*messaging.Message, error) {
		if data, ok := msg.Data.(string); ok {
			received <- data
		}
		return nil, nil
	}

	h.RegisterAgent(agentA, handlerA)
	h.RegisterAgent(agentB, handlerB)

	// Send message
	err := h.Send(ctx, "agent-a", "agent-b", "test-message")
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	// Verify received
	select {
	case data := <-received:
		if data != "test-message" {
			t.Errorf("Received data = %v, want %v", data, "test-message")
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for message")
	}
}

func TestHub_Request(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultHubConfig()
	h := hub.New(ctx, cfg)
	defer h.Shutdown(5 * time.Second)

	agentA := &testAgent{id: "agent-a"}
	agentB := &testAgent{id: "agent-b"}

	handlerA := func(ctx context.Context, msg *messaging.Message, msgCtx *hub.MessageContext) (*messaging.Message, error) {
		return nil, nil
	}

	handlerB := func(ctx context.Context, msg *messaging.Message, msgCtx *hub.MessageContext) (*messaging.Message, error) {
		// Echo back with modification
		if data, ok := msg.Data.(string); ok {
			return messaging.NewResponse("agent-b", msg.From, msg.ID, "processed: "+data).Build(), nil
		}
		return nil, nil
	}

	h.RegisterAgent(agentA, handlerA)
	h.RegisterAgent(agentB, handlerB)

	// Send request
	response, err := h.Request(ctx, "agent-a", "agent-b", "task")
	if err != nil {
		t.Fatalf("Request() error = %v", err)
	}

	if data, ok := response.Data.(string); !ok || data != "processed: task" {
		t.Errorf("Response data = %v, want %v", response.Data, "processed: task")
	}
}
```

## Step 4: Integration Examples

Create examples showing go-agents integration:

```go
// examples/basic/main.go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/JaimeStill/go-agents/pkg/agent"
	"github.com/JaimeStill/go-agents/pkg/config"
	"github.com/JaimeStill/go-agents-orchestration/hub"
	hubconfig "github.com/JaimeStill/go-agents-orchestration/config"
	"github.com/JaimeStill/go-agents-orchestration/messaging"
)

type MyAgent struct {
	id    string
	agent agent.Agent
}

func (a *MyAgent) ID() string { return a.id }

func main() {
	ctx := context.Background()

	// Create go-agents agent
	agentConfig := &config.AgentConfig{
		Transport: &config.TransportConfig{
			Provider: "openai",
			Model:    "gpt-4",
			APIKey:   "your-api-key",
		},
		SystemPrompt: "You are a helpful assistant",
	}

	llmAgent, _ := agent.New(agentConfig)
	myAgent := &MyAgent{id: "assistant", agent: llmAgent}

	// Create hub
	hubCfg := hubconfig.DefaultHubConfig()
	h := hub.New(ctx, hubCfg)
	defer h.Shutdown(5 * time.Second)

	// Register agent with handler
	handler := func(ctx context.Context, msg *messaging.Message, msgCtx *hub.MessageContext) (*messaging.Message, error) {
		// Use go-agents to process message
		response, err := myAgent.agent.Chat(ctx, fmt.Sprintf("Process: %v", msg.Data))
		if err != nil {
			return nil, err
		}

		// Return response message
		return messaging.NewResponse(myAgent.ID(), msg.From, msg.ID, response.Content()).Build(), nil
	}

	h.RegisterAgent(myAgent, handler)

	fmt.Println("Hub and agent initialized successfully")
}
```

## Next Steps

After completing Phase 1:

1. **Validate integration**: Ensure hub works with go-agents
2. **Comprehensive testing**: Achieve 80%+ coverage
3. **Update documentation**: Reflect implemented state
4. **Begin Phase 2**: State graph execution

## Success Criteria for Phase 1

- ✅ messaging/ package complete and tested
- ✅ hub/ package complete and tested
- ✅ All communication patterns work (Send, Request, Broadcast, Pub/Sub)
- ✅ Multi-hub coordination validated
- ✅ go-agents integration demonstrated
- ✅ 80%+ test coverage achieved
- ✅ Documentation complete (README, PROJECT, ARCHITECTURE)
