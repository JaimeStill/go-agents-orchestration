package hub

import (
	"context"

	"github.com/JaimeStill/go-agents/pkg/agent"
	"github.com/JaimeStill/go-agents-orchestration/pkg/messaging"
)

type MessageContext struct {
	HubName string
	Agent   agent.Agent
}

type MessageHandler func(
	ctx context.Context,
	message *messaging.Message,
	context *MessageContext,
) (*messaging.Message, error)
