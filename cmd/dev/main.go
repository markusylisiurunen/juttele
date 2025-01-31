package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	"github.com/markusylisiurunen/juttele"
)

//go:embed prompts/neutral.txt
var neutralSystemPrompt string

func main() {
	var (
		googleToken     = os.Getenv("GOOGLE_TOKEN")
		groqToken       = os.Getenv("GROQ_TOKEN")
		openRouterToken = os.Getenv("OPEN_ROUTER_TOKEN")
	)
	var (
		claude35Sonnet = juttele.NewOpenRouterModel(openRouterToken, "anthropic/claude-3.5-sonnet:beta",
			juttele.WithOpenRouterModelDisplayName("Claude 3.5 Sonnet"),
			juttele.WithOpenRouterModelPersonality("Neutral", neutralSystemPrompt),
		)
		deepseekR1Llama70b = juttele.NewGroqModel(groqToken, "deepseek-r1-distill-llama-70b",
			juttele.WithGroqModelDisplayName("DeepSeek R1 (Llama 70B)"),
			juttele.WithGroqModelPersonality("Neutral", neutralSystemPrompt),
		)
		gemini20FlashThinking = juttele.NewGoogleModel(googleToken, "gemini-2.0-flash-thinking-exp-01-21",
			juttele.WithGoogleModelDisplayName("Gemini 2.0 Flash Thinking"),
			juttele.WithGoogleModelPersonality("Neutral", neutralSystemPrompt),
		)
	)
	app := juttele.New("YOUR_TOKEN_HERE",
		juttele.WithModel(claude35Sonnet),
		juttele.WithModel(deepseekR1Llama70b),
		juttele.WithModel(gemini20FlashThinking),
	)
	if err := app.ListenAndServe(context.Background()); err != nil {
		fmt.Printf("error: %v\n", err)
	}
}
