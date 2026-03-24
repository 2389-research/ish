// ABOUTME: Seed data generation for Anthropic plugin
// ABOUTME: Creates default scenarios for pattern-matched message responses

package anthropic

import (
	"context"

	"github.com/2389/ish/plugins/core"
)

// Seed creates default scenarios for the Anthropic plugin
func (p *AnthropicPlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
	// "hello" pattern scenario with higher priority
	hello := &Scenario{
		Pattern:      "hello",
		ResponseType: "text",
		ResponseText: "Hello! I'm Claude, an AI assistant made by Anthropic. How can I help you today?",
		Priority:     10,
	}
	if err := p.store.CreateScenario(hello); err != nil {
		return core.SeedData{}, err
	}

	// Catch-all scenario with lowest priority (empty pattern matches anything)
	catchAll := &Scenario{
		Pattern:      "",
		ResponseType: "text",
		ResponseText: "I understand your request. Let me help you with that.",
		Priority:     0,
	}
	if err := p.store.CreateScenario(catchAll); err != nil {
		return core.SeedData{}, err
	}

	return core.SeedData{
		Summary: "Created 2 default scenarios (hello greeting + catch-all)",
		Records: map[string]int{
			"scenarios": 2,
		},
	}, nil
}
