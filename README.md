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
