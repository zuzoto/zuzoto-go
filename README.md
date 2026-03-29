# zuzoto-go

The official Go SDK for [Zuzoto](https://github.com/zuzoto/zuzoto) — cognitive memory infrastructure for AI agents.

## Install

```bash
go get github.com/zuzoto/zuzoto-go
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	zuzoto "github.com/zuzoto/zuzoto-go"
)

func main() {
	client := zuzoto.NewClient("http://localhost:8080",
		zuzoto.WithAPIKey("your-api-key"),
	)

	// Add a memory
	result, err := client.Add(context.Background(), &zuzoto.AddInput{
		Content: "User prefers dark mode and uses vim",
		UserID:  "user-123",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Added %d memories, extracted %d entities\n",
		len(result.Memories), result.EntitiesCreated)

	// Search memories
	results, err := client.Search(context.Background(), &zuzoto.SearchQuery{
		Text:   "What editor does the user prefer?",
		UserID: "user-123",
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, m := range results.Memories {
		fmt.Printf("[%.2f] %s\n", m.Score, m.Memory.Content)
	}

	// Get context for LLM
	ctx, err := client.GetContext(context.Background(), &zuzoto.ContextQuery{
		Query:     "Tell me about the user's preferences",
		UserID:    "user-123",
		MaxTokens: 4096,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Context: %d memories, %d facts, %d tokens\n",
		len(ctx.Memories), len(ctx.Facts), ctx.Tokens)
}
```

## Features

- **Memory CRUD** — add, get, update, search, forget
- **Batch ingestion** — add up to 100 memories in one call
- **Hybrid search** — vector, BM25, graph, temporal strategies
- **Context assembly** — token-budgeted context windows for LLMs
- **Knowledge graph** — entity and fact CRUD, temporal state queries
- **Sessions** — conversation session management
- **Temporal queries** — point-in-time and entity timeline queries

## License

Apache 2.0
