# Juttele

```go
package main

func main() {
  app := juttele.New(
    "YOUR_TOKEN_HERE",
    juttele.WithModel(
      juttele.NewOpenRouterModel("OPEN_ROUTER_TOKEN", "anthropic/claude-3.5-sonnet:beta",
        juttele.WithOpenRouterModelPersonality("Neutral", "You are a helpful and friendly AI."),
      ),
    ),
  )
  if err := app.ListenAndServe(context.Background()); err != nil {
    fmt.Printf("error: %v\n", err)
  }
}
```

```bash
curl -s -X GET \
  -H 'Authorization: Bearer YOUR_TOKEN_HERE' \
  localhost:8765/models
```

```json
{
  "models": [
    {
      "id": "openrouter_13418125828435531457",
      "name": "Claude 3.5 Sonnet",
      "personalities": [
        {
          "id": "13063254767047260133",
          "name": "Neutral"
        }
      ]
    },
    {
      "id": "groq_3185326761375271060",
      "name": "DeepSeek R1 (Llama 70B)",
      "personalities": [
        {
          "id": "2997477578727550792",
          "name": "Neutral"
        }
      ]
    },
    {
      "id": "google_3533711261352199707",
      "name": "Gemini 2.0 Flash Thinking",
      "personalities": [
        {
          "id": "7145165113051147527",
          "name": "Neutral"
        }
      ]
    }
  ]
}
```

```bash
curl -s -X POST \
  -H 'Authorization: Bearer YOUR_TOKEN_HERE' \
  -d '{
    "model_id": "openrouter_13418125828435531457",
    "model_personality_id": "13063254767047260133",
    "history": [{"role": "user", "content": "hey :)"}]
  }' \
  localhost:8765/stream
```

```plaintext
data: {"content":""}

data: {"content":""}

data: {"content":"Hi"}

data: {"content":" there! How are you today? "}

data: {"content":"😊"}

data: {"content":""}

data: {"content":""}
```
