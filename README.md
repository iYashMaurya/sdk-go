# Lingo.dev Go SDK

[![Go Report Card](https://goreportcard.com/badge/github.com/lingodotdev/sdk-go)](https://goreportcard.com/report/github.com/lingodotdev/sdk-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/lingodotdev/sdk-go.svg)](https://pkg.go.dev/github.com/lingodotdev/sdk-go)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

A production-grade Go SDK for the Lingo.dev localization engine. Supports text, objects, chat messages, batch operations, and language detection with built-in concurrency, automatic retries, and full context propagation.

## ✨ Key Features

- 🧵 **Context-aware** — every method accepts `context.Context` for cancellation and timeouts
- 🔀 **Concurrent processing** with `errgroup` for dramatically faster bulk translations
- 🛡️ **Typed errors** — `ValueError` and `RuntimeError` catchable with `errors.As`
- 🔁 **Automatic retries** with exponential backoff (1s, 2s, 4s) on 5xx errors and network failures
- 🎯 **Multiple content types** — text, key-value objects, chat messages
- 🌐 **Auto-detection** of source languages
- ⚡ **Fast mode** for quick translations when precision is less critical
- � **Functional options** pattern for clean, extensible configuration

## 🚀 Performance Benefits

The concurrent implementation provides significant performance improvements:

- **Parallel chunk processing** — large payloads are split and translated simultaneously
- **Batch operations** — translate to multiple languages or multiple objects in one call
- **Concurrent API requests** via `errgroup` instead of sequential loops
- **Context cancellation** propagated to the HTTP layer — cancelled requests stop immediately

## 📦 Installation

```bash
go get github.com/lingodotdev/sdk-go
```

## 🎯 Quick Start

### Simple Translation

```go
package main

import (
	"context"
	"fmt"
	"log"

	lingo "github.com/lingodotdev/sdk-go"
)

func main() {
	client, err := lingo.NewClient("your-api-key")
	if err != nil {
		log.Fatal(err)
	}

	result, err := client.LocalizeText(
		context.Background(),
		"Hello, world!",
		lingo.LocalizationParams{TargetLocale: "es"},
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result) // "¡Hola, mundo!"
}
```

### With Source Locale and Fast Mode

`SourceLocale` and `Fast` are pointer types — use `&` to pass optional values:

```go
package main

import (
	"context"
	"fmt"
	"log"

	lingo "github.com/lingodotdev/sdk-go"
)

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

func main() {
	client, err := lingo.NewClient("your-api-key")
	if err != nil {
		log.Fatal(err)
	}

	result, err := client.LocalizeText(
		context.Background(),
		"Hello, world!",
		lingo.LocalizationParams{
			SourceLocale: strPtr("en"),
			TargetLocale: "es",
			Fast:         boolPtr(true),
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result)
}
```

## 🔥 Advanced Usage

### Object Translation

```go
package main

import (
	"context"
	"fmt"
	"log"

	lingo "github.com/lingodotdev/sdk-go"
)

func main() {
	client, err := lingo.NewClient("your-api-key")
	if err != nil {
		log.Fatal(err)
	}

	obj := map[string]any{
		"greeting": "Hello",
		"farewell": "Goodbye",
		"question": "How are you?",
	}

	result, err := client.LocalizeObject(
		context.Background(),
		obj,
		lingo.LocalizationParams{TargetLocale: "fr"},
		true, // concurrent — process chunks in parallel for speed
	)
	if err != nil {
		log.Fatal(err)
	}

	for k, v := range result {
		fmt.Printf("%s: %v\n", k, v)
	}
}
```

### Batch Translation (Multiple Target Languages)

```go
package main

import (
	"context"
	"fmt"
	"log"

	lingo "github.com/lingodotdev/sdk-go"
)

func main() {
	client, err := lingo.NewClient("your-api-key")
	if err != nil {
		log.Fatal(err)
	}

	results, err := client.BatchLocalizeText(
		context.Background(),
		"Welcome to our application",
		nil, // sourceLocale (auto-detect)
		nil, // fast (default)
		[]string{"es", "fr", "de", "it"},
	)
	if err != nil {
		log.Fatal(err)
	}

	for i, result := range results {
		fmt.Printf("[%d] %s\n", i, result)
	}
}
```

### Chat Translation

Speaker names are preserved while text is translated:

```go
package main

import (
	"context"
	"fmt"
	"log"

	lingo "github.com/lingodotdev/sdk-go"
)

func strPtr(s string) *string { return &s }

func main() {
	client, err := lingo.NewClient("your-api-key")
	if err != nil {
		log.Fatal(err)
	}

	chat := []map[string]string{
		{"name": "Alice", "text": "Hello everyone!"},
		{"name": "Bob", "text": "How is everyone doing?"},
		{"name": "Charlie", "text": "Great to see you all!"},
	}

	translated, err := client.LocalizeChat(
		context.Background(),
		chat,
		lingo.LocalizationParams{
			SourceLocale: strPtr("en"),
			TargetLocale: "es",
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	for _, msg := range translated {
		fmt.Printf("%s: %s\n", msg["name"], msg["text"])
	}
}
```

### Batch Object Translation

```go
package main

import (
	"context"
	"fmt"
	"log"

	lingo "github.com/lingodotdev/sdk-go"
)

func main() {
	client, err := lingo.NewClient("your-api-key")
	if err != nil {
		log.Fatal(err)
	}

	objects := []map[string]any{
		{"title": "Welcome", "description": "Please sign in"},
		{"error": "Invalid input", "help": "Check your email"},
		{"success": "Account created", "next": "Continue to dashboard"},
	}

	results, err := client.BatchLocalizeObjects(
		context.Background(),
		objects,
		lingo.LocalizationParams{TargetLocale: "fr"},
	)
	if err != nil {
		log.Fatal(err)
	}

	for i, result := range results {
		fmt.Printf("Object %d: %v\n", i, result)
	}
}
```

### Language Detection

```go
package main

import (
	"context"
	"fmt"
	"log"

	lingo "github.com/lingodotdev/sdk-go"
)

func main() {
	client, err := lingo.NewClient("your-api-key")
	if err != nil {
		log.Fatal(err)
	}

	locale, err := client.RecognizeLocale(context.Background(), "Bonjour le monde")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(locale) // "fr"
}
```

### Account Info

```go
package main

import (
	"context"
	"fmt"
	"log"

	lingo "github.com/lingodotdev/sdk-go"
)

func main() {
	client, err := lingo.NewClient("your-api-key")
	if err != nil {
		log.Fatal(err)
	}

	info, err := client.WhoAmI(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	if info == nil {
		fmt.Println("Not authenticated")
		return
	}

	for k, v := range info {
		fmt.Printf("%s: %s\n", k, v)
	}
}
```

### Context and Cancellation

Go's `context.Context` propagates all the way to the HTTP layer. Use timeouts and cancellation to control long-running operations:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	lingo "github.com/lingodotdev/sdk-go"
)

func main() {
	client, err := lingo.NewClient("your-api-key")
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := client.LocalizeText(
		ctx,
		"Hello, world!",
		lingo.LocalizationParams{TargetLocale: "es"},
	)
	if err != nil {
		log.Fatalf("translation failed or timed out: %s", err)
	}

	fmt.Println(result)
}
```

If the context is cancelled or times out, in-flight HTTP requests are aborted immediately — no wasted network calls.

## ⚙️ Configuration Options

```go
client, err := lingo.NewClient(
	"your-api-key",
	lingo.SetURL("https://engine.lingo.dev"),  // Optional: API endpoint (default shown)
	lingo.SetBatchSize(25),                     // Optional: Items per chunk (1–250, default 25)
	lingo.SetIdealBatchItemSize(250),           // Optional: Target words per chunk (1–2500, default 250)
)
```

| Option | Default | Range | Description |
|--------|---------|-------|-------------|
| `SetURL` | `https://engine.lingo.dev` | Valid HTTP/HTTPS URL | API endpoint |
| `SetBatchSize` | `25` | 1–250 | Maximum items per chunk before splitting |
| `SetIdealBatchItemSize` | `250` | 1–2500 | Target word count per chunk before splitting |

## 🎛️ LocalizationParams Reference

```go
type LocalizationParams struct {
	SourceLocale *string                   // Source language code (nil = auto-detect)
	TargetLocale string                    // Target language code (required)
	Fast         *bool                     // Enable fast mode for quicker translations (nil = default)
	Reference    map[string]map[string]any // Reference translations for additional context
}
```

Optional fields use pointer types (`*string`, `*bool`) so that `nil` means "use the default" rather than requiring a zero value. Use small helpers to pass values inline:

```go
func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }
```

## 📚 API Reference

### Core Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| **LocalizeText** | `(ctx context.Context, text string, params LocalizationParams) (string, error)` | Translate a single text string |
| **LocalizeObject** | `(ctx context.Context, obj map[string]any, params LocalizationParams, concurrent bool) (map[string]any, error)` | Translate all values in a map |
| **LocalizeChat** | `(ctx context.Context, chat []map[string]string, params LocalizationParams) ([]map[string]string, error)` | Translate chat messages, preserving speaker names |
| **RecognizeLocale** | `(ctx context.Context, text string) (string, error)` | Detect the language of a text string |
| **WhoAmI** | `(ctx context.Context) (map[string]string, error)` | Get authenticated user info; returns `nil, nil` if unauthenticated |

### Batch Methods

| Method | Signature | Description |
|--------|-----------|-------------|
| **BatchLocalizeText** | `(ctx context.Context, text string, sourceLocale *string, fast *bool, targetLocales []string) ([]string, error)` | Translate text to multiple languages concurrently |
| **BatchLocalizeObjects** | `(ctx context.Context, objects []map[string]any, params LocalizationParams) ([]map[string]any, error)` | Translate multiple objects concurrently |

## 🔧 Error Handling

The SDK uses two typed errors that you can catch with `errors.As`:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	lingo "github.com/lingodotdev/sdk-go"
)

func main() {
	client, err := lingo.NewClient("your-api-key")
	if err != nil {
		log.Fatal(err)
	}

	_, err = client.LocalizeText(
		context.Background(),
		"Hello",
		lingo.LocalizationParams{TargetLocale: "es"},
	)
	if err != nil {
		var ve *lingo.ValueError
		if errors.As(err, &ve) {
			// Validation error: empty text, invalid config, bad request (400)
			fmt.Printf("ValueError: %s\n", ve.Message)
			return
		}

		var re *lingo.RuntimeError
		if errors.As(err, &re) {
			// Runtime error: server error (5xx), network failure, unexpected response
			fmt.Printf("RuntimeError: %s (status %d)\n", re.Message, re.StatusCode)
			return
		}

		log.Fatal(err)
	}
}
```

| Error Type | Returned When |
|------------|---------------|
| `*ValueError` | Empty text, invalid config options, API returns 400 Bad Request |
| `*RuntimeError` | Server errors (5xx), network failures, unexpected response format, JSON parse errors |

`RuntimeError.StatusCode` contains the HTTP status code when the error originated from an HTTP response (0 otherwise).

## 🚀 Performance Tips

1. **Use `concurrent: true`** in `LocalizeObject` for large payloads — chunks are processed in parallel via `errgroup`
2. **Use `BatchLocalizeText`** to translate one text into multiple languages concurrently instead of sequential `LocalizeText` calls
3. **Use `BatchLocalizeObjects`** to translate multiple objects at once — each object is processed in its own goroutine
4. **Tune `SetBatchSize`** based on your content — smaller batches create more chunks which enables more parallelism
5. **Use `context.WithTimeout`** to set hard deadlines — the context propagates to the HTTP layer and cancels in-flight requests
6. **Let the SDK retry** — 5xx errors and network failures are retried automatically up to 3 times with exponential backoff; don't add your own retry wrapper

## 🧪 Running Tests

This project uses a Makefile for common tasks.

### Unit Tests

```bash
make test
```

### Integration Tests (requires API key)

```bash
export LINGODOTDEV_API_KEY="your-api-key"
make test-integration
```

### With Coverage Report

```bash
make test-coverage
```

Opens an HTML coverage report in your browser automatically.

### All Available Commands

```bash
make help
```

## 🏆 Go Advantages

The Go SDK offers several advantages over other language implementations:

- **Native context cancellation** — `context.Context` propagates through every layer down to `http.NewRequestWithContext`, so cancelled requests are aborted at the HTTP transport level with zero wasted network calls
- **Zero external dependencies for core functionality** — the only dependencies are `golang.org/x/sync` for `errgroup` and `go-nanoid` for workflow IDs; the core HTTP, JSON, and retry logic uses only the standard library
- **Typed errors with `errors.As`** — no string matching or catch-all exception handlers; `ValueError` and `RuntimeError` are distinct types you can match precisely
- **No async/await complexity** — standard synchronous function calls with goroutines and `errgroup` for concurrency; no colored functions, no event loop, no runtime overhead

## 📌 Versioning

This SDK follows [Semantic Versioning](https://semver.org/). Check the [releases page](https://github.com/lingodotdev/sdk-go/releases) for the latest version.

## 📄 License

Apache-2.0 License

## 🤖 Support

- 📚 [Documentation](https://lingo.dev/docs)
- 🐛 [Issues](https://github.com/lingodotdev/sdk-go/issues)
- 💬 [Community](https://lingo.dev/discord)
