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
	var (
		claude37Sonnet = juttele.NewAnthropicModel(anthropicToken, "claude-3-7-sonnet-20250219",
			juttele.WithAnthropicModelDisplayName("Claude 3.7 Sonnet"),
			juttele.WithAnthropicModelPersonality("Raw", rawSystemPrompt),
		)
		claude37SonnetThinking = juttele.NewAnthropicModel(anthropicToken, "claude-3-7-sonnet-20250219",
			juttele.WithAnthropicModelDisplayName("Claude 3.7 Sonnet (thinking)"),
			juttele.WithAnthropicModelPersonality("Raw", rawSystemPrompt),
			juttele.WithAnthropicModelExtendedThinking(),
		)
		gpt4o = juttele.NewOpenRouterModel(openRouterToken, "openai/gpt-4o-2024-11-20",
			juttele.WithOpenRouterModelDisplayName("GPT-4o"),
			juttele.WithOpenRouterModelPersonality("Raw", rawSystemPrompt),
			juttele.WithOpenRouterModelTools(juttele.NewMemoryToolBundle("./.data")...),
		)
		deepseekR1 = juttele.NewDeepSeekModel(deepSeekToken, "deepseek-reasoner",
			juttele.WithDeepSeekModelDisplayName("DeepSeek R1"),
			juttele.WithDeepSeekModelPersonality("Raw", rawSystemPrompt),
		)
		deepseekR1Llama70b = juttele.NewGroqModel(groqToken, "deepseek-r1-distill-llama-70b",
			juttele.WithGroqModelDisplayName("DeepSeek R1 (Llama 70B)"),
			juttele.WithGroqModelPersonality("Raw", rawSystemPrompt),
		)
		gemini20FlashThinking = juttele.NewGoogleModel(googleToken, "gemini-2.0-flash-thinking-exp-01-21",
			juttele.WithGoogleModelDisplayName("Gemini 2.0 Flash Thinking"),
			juttele.WithGoogleModelPersonality("Raw", rawSystemPrompt),
		)
	)
	app := juttele.New("YOUR_TOKEN_HERE",
		juttele.WithModel(claude37Sonnet),
		juttele.WithModel(claude37SonnetThinking),
		juttele.WithModel(gpt4o),
		juttele.WithModel(deepseekR1),
		juttele.WithModel(deepseekR1Llama70b),
		juttele.WithModel(gemini20FlashThinking),
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
