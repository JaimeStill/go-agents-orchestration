# go-agents-orchestration

Go-native agent coordination primitives for building multi-agent systems with LangGraph-inspired state management and composable workflow patterns.

## Overview

**go-agents-orchestration** is a supplemental package in the [go-agents](https://github.com/JaimeStill/go-agents) ecosystem that provides coordination and orchestration capabilities for Go-based LLM agent applications.

### Relationship to go-agents

- **Parent Library**: [github.com/JaimeStill/go-agents](https://github.com/JaimeStill/go-agents) (v0.1.0)
- **Purpose**: Extends go-agents with multi-agent coordination, state management, and workflow orchestration
- **Integration**: Compose go-agents `Agent` interface with orchestration primitives

### What This Package Provides

- **Hub Coordination**: Multi-hub agent networking with message routing and cross-hub communication
- **Messaging Primitives**: Structured inter-agent messaging with send, request/response, broadcast, and pub/sub patterns
- **State Management**: LangGraph-inspired state graph execution with transitions, predicates, and checkpointing
- **Workflow Patterns**: Sequential chains, parallel execution, conditional routing, and stateful workflows
- **Observability**: Execution tracing, decision logging, and performance metrics

### What This Package Does NOT Provide

Capabilities intentionally left to the go-agents library:

- LLM protocol execution (Chat, Vision, Tools, Embeddings)
- Provider integration (OpenAI, Azure, Ollama, etc.)
- HTTP transport and streaming
- Capability format system

## Design Philosophy

Following go-agents principles:

1. **Minimal Abstractions**: Essential primitives for agent coordination
2. **Format Extensibility**: Enable new patterns without modifying core code
3. **Configuration-Driven**: Initialize infrastructure through configuration
4. **Type Safety**: Leverage Go's type system for compile-time safety
5. **Go-Native Patterns**: Embrace Go concurrency idioms (channels, contexts, goroutines)

## Development Status

This package is under active development and follows pre-release versioning (v0.x.x). Breaking changes may occur during the validation period. The package will graduate to v1.0.0 after validating go-agents integration patterns and achieving production readiness.

**Development Phases:**
- **Phase 1**: Foundation (messaging + hub coordination) - ✅ **Complete**
- **Phase 2**: State graph execution - Planned
- **Phase 3**: Workflow patterns - Planned
- **Phase 4**: Observability infrastructure - Planned

## Documentation

- **[PROJECT.md](./PROJECT.md)**: Project scope, roadmap, and success criteria
- **[ARCHITECTURE.md](./ARCHITECTURE.md)**: Technical specifications and design patterns
- **[CLAUDE.md](./CLAUDE.md)**: Development guidelines and design principles
- **[_context/](./_context/)**: Implementation guides and development summaries

## Examples

### [Phase 1: Hub & Messaging](./examples/phase-01-hubs)

ISS Maintenance EVA scenario demonstrating all Phase 1 orchestration capabilities:
- Agent-to-agent communication (direct messaging between agents)
- Broadcast communication (one-to-many within a hub)
- Pub/sub messaging (topic-based distribution with sender filtering)
- Cross-hub coordination (agents registered in multiple hubs)


## Installation

```bash
go get github.com/JaimeStill/go-agents-orchestration
```

**Requirements:**
- Go 1.23 or later
- [go-agents](https://github.com/JaimeStill/go-agents) v0.1.2+

## Quick Start

### Basic Hub Setup

```go
package main

import (
    "context"
    "time"

    "github.com/JaimeStill/go-agents-orchestration/pkg/config"
    "github.com/JaimeStill/go-agents-orchestration/pkg/hub"
    "github.com/JaimeStill/go-agents-orchestration/pkg/messaging"
    "github.com/JaimeStill/go-agents/pkg/agent"
)

func main() {
    ctx := context.Background()

    // Create hub
    hubConfig := config.DefaultHubConfig()
    hubConfig.Name = "main-hub"
    h := hub.New(ctx, hubConfig)
    defer h.Shutdown(5 * time.Second)

    // Create and register agents
    agent1, _ := agent.New(agentConfig1)
    agent2, _ := agent.New(agentConfig2)

    handler1 := func(ctx context.Context, msg *messaging.Message, msgCtx *hub.MessageContext) (*messaging.Message, error) {
        // Process message with LLM
        response, _ := agent1.Chat(ctx, msg.Data.(string))
        return messaging.NewResponse(agent1.ID(), msg.From, msg.ID, response.Content()).Build(), nil
    }

    h.RegisterAgent(agent1, handler1)
    h.RegisterAgent(agent2, handler2)

    // Send messages
    h.Send(ctx, agent1.ID(), agent2.ID(), "Hello!")
}
```

## Communication Patterns

### Send (Fire-and-Forget)
```go
h.Send(ctx, fromAgentID, toAgentID, data)
```

### Request/Response
```go
response, err := h.Request(ctx, fromAgentID, toAgentID, data)
```

### Broadcast
```go
h.Broadcast(ctx, fromAgentID, data) // Sends to all except sender
```

### Pub/Sub
```go
h.Subscribe(agentID, "topic-name")
h.Publish(ctx, fromAgentID, "topic-name", data) // Sender filtered out
```

