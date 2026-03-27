package main

import (
	"context"
	"fmt"
	"log"
	"os"

	lingo "github.com/lingodotdev/sdk-go"
)

func main() {
	apiKey := os.Getenv("LINGODOTDEV_API_KEY")
	if apiKey == "" {
		fmt.Println("LINGODOTDEV_API_KEY environment variable is not set.")
		fmt.Println("Set it with: export LINGODOTDEV_API_KEY=\"your-api-key\"")
		os.Exit(1)
	}

	client, err := lingo.NewClient(apiKey)
	if err != nil {
		log.Fatalf("failed to create client: %s", err)
	}

	ctx := context.Background()

	// --- LocalizeText ---
	fmt.Println("=== LocalizeText ===")
	translated, err := client.LocalizeText(ctx, "Hello, world!", lingo.LocalizationParams{
		TargetLocale: "es",
	})
	if err != nil {
		log.Fatalf("LocalizeText error: %s", err)
	}
	fmt.Printf("Translated: %s\n\n", translated)

	// --- LocalizeObject ---
	fmt.Println("=== LocalizeObject ===")
	obj := map[string]any{
		"greeting": "Hello",
		"farewell": "Goodbye",
		"question": "How are you?",
	}
	objResult, err := client.LocalizeObject(ctx, obj, lingo.LocalizationParams{
		TargetLocale: "fr",
	}, true)
	if err != nil {
		log.Fatalf("LocalizeObject error: %s", err)
	}
	for k, v := range objResult {
		fmt.Printf("  %s: %v\n", k, v)
	}
	fmt.Println()

	// --- BatchLocalizeText ---
	fmt.Println("=== BatchLocalizeText ===")
	locales := []string{"es", "fr", "de"}
	batchResults, err := client.BatchLocalizeText(ctx, "Welcome to our application", nil, nil, locales)
	if err != nil {
		log.Fatalf("BatchLocalizeText error: %s", err)
	}
	for i, result := range batchResults {
		fmt.Printf("  %s: %s\n", locales[i], result)
	}
	fmt.Println()

	// --- LocalizeChat ---
	fmt.Println("=== LocalizeChat ===")
	chat := []map[string]string{
		{"name": "Alice", "text": "Hello everyone!"},
		{"name": "Bob", "text": "How are you?"},
	}
	chatResult, err := client.LocalizeChat(ctx, chat, lingo.LocalizationParams{
		TargetLocale: "es",
	})
	if err != nil {
		log.Fatalf("LocalizeChat error: %s", err)
	}
	for _, msg := range chatResult {
		fmt.Printf("  %s: %s\n", msg["name"], msg["text"])
	}
	fmt.Println()

	// --- RecognizeLocale ---
	fmt.Println("=== RecognizeLocale ===")
	locale, err := client.RecognizeLocale(ctx, "Bonjour le monde")
	if err != nil {
		log.Fatalf("RecognizeLocale error: %s", err)
	}
	fmt.Printf("Detected locale: %s\n\n", locale)

	// --- WhoAmI ---
	fmt.Println("=== WhoAmI ===")
	info, err := client.WhoAmI(ctx)
	if err != nil {
		log.Fatalf("WhoAmI error: %s", err)
	}
	if info == nil {
		fmt.Println("Not authenticated or no user info available.")
	} else {
		for k, v := range info {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}
}
