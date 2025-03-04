package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/markusylisiurunen/juttele"
)

//go:embed prompts/raw.txt
var rawSystemPrompt string

func main() {
	var (
		anthropicToken  = os.Getenv("ANTHROPIC_TOKEN")
		deepSeekToken   = os.Getenv("DEEPSEEK_TOKEN")
		googleToken     = os.Getenv("GOOGLE_TOKEN")
		groqToken       = os.Getenv("GROQ_TOKEN")
		openRouterToken = os.Getenv("OPEN_ROUTER_TOKEN")
	)
	app := juttele.New("YOUR_TOKEN_HERE",
		juttele.WithModel(
			juttele.NewAnthropicModel(anthropicToken, "claude-3-7-sonnet-20250219", false,
				juttele.WithDisplayName("Claude 3.7 Sonnet (standard)"),
				juttele.WithMaxTokens(16384),
				juttele.WithPersonality("Raw", rawSystemPrompt),
				juttele.WithTemperature(0.7),
			),
		),
		juttele.WithModel(
			juttele.NewAnthropicModel(anthropicToken, "claude-3-7-sonnet-20250219", true,
				juttele.WithDisplayName("Claude 3.7 Sonnet (thinking)"),
				juttele.WithMaxTokens(16384),
				juttele.WithPersonality("Raw", rawSystemPrompt),
				juttele.WithTemperature(1.0),
			),
		),
		juttele.WithModel(
			juttele.NewDeepSeekModel(deepSeekToken, "deepseek-reasoner",
				juttele.WithDisplayName("DeepSeek R1"),
				juttele.WithMaxTokens(8192),
				juttele.WithPersonality("Raw", rawSystemPrompt),
				juttele.WithTemperature(0.6),
			),
		),
		juttele.WithModel(
			juttele.NewOpenRouterModel(openRouterToken, "openai/gpt-4o-2024-11-20",
				juttele.WithDisplayName("GPT-4o"),
				juttele.WithMaxTokens(16384),
				juttele.WithPersonality("Raw", rawSystemPrompt),
				juttele.WithTemperature(0.7),
			),
		),
		juttele.WithModel(
			juttele.NewOpenRouterModel(openRouterToken, "openai/o3-mini-high",
				juttele.WithDisplayName("o3-mini (high)"),
				juttele.WithMaxTokens(16384),
				juttele.WithPersonality("Raw", rawSystemPrompt),
				juttele.WithTemperature(0.7),
			),
		),
		juttele.WithModel(
			juttele.NewGroqModel(groqToken, "deepseek-r1-distill-llama-70b",
				juttele.WithDisplayName("DeepSeek R1 (Llama 70B)"),
				juttele.WithMaxTokens(8192),
				juttele.WithPersonality("Raw", rawSystemPrompt),
				juttele.WithTemperature(0.6),
			),
		),
		juttele.WithModel(
			juttele.NewGoogleModel(googleToken, "gemini-2.0-flash-thinking-exp",
				juttele.WithDisplayName("Gemini 2.0 Flash Thinking"),
				juttele.WithMaxTokens(8192),
				juttele.WithPersonality("Raw", rawSystemPrompt),
				juttele.WithTemperature(1.0),
			),
		),
		juttele.WithToolBundle(juttele.NewMemoryToolBundle("./.data")),
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := app.ListenAndServe(ctx); err != nil {
			fmt.Printf("error: %v\n", err)
		}
	}()
	<-c
	cancel()
	<-done
}
