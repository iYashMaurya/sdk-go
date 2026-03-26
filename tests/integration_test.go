package tests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	lingo "github.com/lingodotdev/sdk-go"
)

// --- Helper functions ---

func skipIfNoAPIKey(t *testing.T) (string, string) {
	t.Helper()
	apiKey := os.Getenv("LINGODOTDEV_API_KEY")
	if apiKey == "" {
		t.Skip("LINGODOTDEV_API_KEY not set, skipping real API test")
	}
	apiURL := os.Getenv("LINGODOTDEV_API_URL")
	if apiURL == "" {
		apiURL = "https://engine.lingo.dev"
	}
	return apiKey, apiURL
}

func newRealClient(t *testing.T) *lingo.Client {
	t.Helper()
	apiKey, apiURL := skipIfNoAPIKey(t)
	client, err := lingo.NewClient(apiKey, lingo.SetURL(apiURL))
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}
	return client
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

// --- GROUP 1: Real API Tests ---

func TestRealAPI_LocalizeText(t *testing.T) {
	client := newRealClient(t)

	params := lingo.LocalizationParams{
		SourceLocale: strPtr("en"),
		TargetLocale: "es",
	}
	result, err := client.LocalizeText(context.Background(), "Hello, world!", params)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	if result == "Hello, world!" {
		t.Fatal("expected translated text, got original")
	}
	t.Logf("LocalizeText result: %s", result)
}

func TestRealAPI_LocalizeObject(t *testing.T) {
	client := newRealClient(t)

	obj := map[string]any{
		"greeting": "Hello",
		"farewell": "Goodbye",
		"question": "How are you?",
	}
	params := lingo.LocalizationParams{
		SourceLocale: strPtr("en"),
		TargetLocale: "fr",
	}
	result, err := client.LocalizeObject(context.Background(), obj, params, false)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(result))
	}
	for _, key := range []string{"greeting", "farewell", "question"} {
		val, ok := result[key]
		if !ok {
			t.Errorf("missing key %q in result", key)
			continue
		}
		valStr, ok := val.(string)
		if !ok {
			t.Errorf("expected string for key %q, got %T", key, val)
			continue
		}
		if valStr == obj[key] {
			t.Errorf("expected translated value for key %q, got original: %s", key, valStr)
		}
	}
	t.Logf("LocalizeObject result: %v", result)
}

func TestRealAPI_BatchLocalizeText(t *testing.T) {
	client := newRealClient(t)

	original := "Welcome to our application"
	locales := []string{"es", "fr", "de"}
	results, err := client.BatchLocalizeText(context.Background(), original, strPtr("en"), nil, locales)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for i, result := range results {
		if result == "" {
			t.Errorf("locale %s: expected non-empty result", locales[i])
		}
		if result == original {
			t.Errorf("locale %s: expected translated text, got original", locales[i])
		}
	}
	t.Logf("BatchLocalizeText results: %v", results)
}

func TestRealAPI_LocalizeChat(t *testing.T) {
	client := newRealClient(t)

	chat := []map[string]string{
		{"name": "Alice", "text": "Hello everyone!"},
		{"name": "Bob", "text": "How are you doing?"},
		{"name": "Charlie", "text": "I'm doing great, thanks!"},
	}
	params := lingo.LocalizationParams{
		SourceLocale: strPtr("en"),
		TargetLocale: "es",
	}
	result, err := client.LocalizeChat(context.Background(), chat, params)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(result))
	}
	expectedNames := []string{"Alice", "Bob", "Charlie"}
	for i, msg := range result {
		name, ok := msg["name"]
		if !ok {
			t.Errorf("message %d: missing 'name' key", i)
			continue
		}
		text, ok := msg["text"]
		if !ok {
			t.Errorf("message %d: missing 'text' key", i)
			continue
		}
		if name != expectedNames[i] {
			t.Errorf("message %d: expected name %q, got %q", i, expectedNames[i], name)
		}
		if text == chat[i]["text"] {
			t.Errorf("message %d: expected translated text, got original: %s", i, text)
		}
	}
	t.Logf("LocalizeChat results: %v", result)
}

func TestRealAPI_RecognizeLocale(t *testing.T) {
	client := newRealClient(t)

	tests := []struct {
		name string
		text string
	}{
		{name: "english", text: "Hello, how are you?"},
		{name: "spanish", text: "Hola, ¿cómo estás?"},
		{name: "french", text: "Bonjour, comment allez-vous?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			locale, err := client.RecognizeLocale(context.Background(), tt.text)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if locale == "" {
				t.Fatal("expected non-empty locale")
			}
			t.Logf("RecognizeLocale(%q) = %s", tt.text, locale)
		})
	}
}

func TestRealAPI_WhoAmI(t *testing.T) {
	client := newRealClient(t)

	result, err := client.WhoAmI(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if result == nil {
		t.Log("WhoAmI returned nil (unauthenticated or no user info)")
		return
	}
	email, ok := result["email"]
	if !ok {
		t.Error("expected 'email' field in result")
	} else if !strings.Contains(email, "@") {
		t.Errorf("expected email to contain '@', got: %s", email)
	}
	if _, ok := result["id"]; !ok {
		t.Error("expected 'id' field in result")
	}
	t.Logf("WhoAmI result: %v", result)
}

func TestRealAPI_FastMode(t *testing.T) {
	client := newRealClient(t)

	params := lingo.LocalizationParams{
		SourceLocale: strPtr("en"),
		TargetLocale: "es",
		Fast:         boolPtr(true),
	}
	fastResult, err := client.LocalizeText(context.Background(), "Hello, world!", params)
	if err != nil {
		t.Fatalf("fast mode error: %s", err)
	}

	params.Fast = boolPtr(false)
	normalResult, err := client.LocalizeText(context.Background(), "Hello, world!", params)
	if err != nil {
		t.Fatalf("normal mode error: %s", err)
	}

	if fastResult == "" {
		t.Error("fast mode returned empty result")
	}
	if normalResult == "" {
		t.Error("normal mode returned empty result")
	}
	if fastResult == "Hello, world!" {
		t.Error("fast mode returned original text")
	}
	if normalResult == "Hello, world!" {
		t.Error("normal mode returned original text")
	}
	t.Logf("Fast result: %s", fastResult)
	t.Logf("Normal result: %s", normalResult)
}

func TestRealAPI_ConcurrentVsSequential(t *testing.T) {
	apiKey, apiURL := skipIfNoAPIKey(t)
	client, err := lingo.NewClient(apiKey, lingo.SetURL(apiURL), lingo.SetBatchSize(2))
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	payload := make(map[string]any)
	for i := 0; i < 10; i++ {
		payload[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("Test content number %d for performance testing", i)
	}

	params := lingo.LocalizationParams{
		SourceLocale: strPtr("en"),
		TargetLocale: "es",
	}

	start := time.Now()
	_, err = client.LocalizeObject(context.Background(), payload, params, false)
	if err != nil {
		t.Fatalf("sequential error: %s", err)
	}
	seqDuration := time.Since(start)

	start = time.Now()
	_, err = client.LocalizeObject(context.Background(), payload, params, true)
	if err != nil {
		t.Fatalf("concurrent error: %s", err)
	}
	concDuration := time.Since(start)

	threshold := time.Duration(float64(seqDuration) * 1.5)
	if concDuration > threshold {
		t.Errorf("concurrent (%s) was not faster than sequential (%s) * 1.5", concDuration, seqDuration)
	}
	t.Logf("Sequential duration: %s", seqDuration)
	t.Logf("Concurrent duration: %s", concDuration)
}

func TestRealAPI_BatchLocalizeObjects(t *testing.T) {
	client := newRealClient(t)

	objects := []map[string]any{
		{"greeting": "Hello", "question": "How are you?"},
		{"farewell": "Goodbye", "thanks": "Thank you"},
		{"welcome": "Welcome", "help": "Can I help you?"},
	}
	params := lingo.LocalizationParams{
		SourceLocale: strPtr("en"),
		TargetLocale: "es",
	}
	results, err := client.BatchLocalizeObjects(context.Background(), objects, params)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for i, result := range results {
		for key, origVal := range objects[i] {
			val, ok := result[key]
			if !ok {
				t.Errorf("object %d: missing key %q", i, key)
				continue
			}
			valStr, ok := val.(string)
			if !ok {
				t.Errorf("object %d: expected string for key %q, got %T", i, key, val)
				continue
			}
			if valStr == origVal {
				t.Errorf("object %d: expected translated value for key %q, got original: %s", i, key, valStr)
			}
		}
	}
	t.Logf("BatchLocalizeObjects results: %v", results)
}

// --- GROUP 2: Mocked Integration Tests ---

func TestMocked_LargePayloadChunking(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		data, ok := body["data"].(map[string]any)
		if !ok {
			data = map[string]any{"key": "value"}
		}
		response := make(map[string]any)
		for k := range data {
			response[k] = "translated"
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": response})
	}))
	defer server.Close()

	client, err := lingo.NewClient("test-key", lingo.SetURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	payload := make(map[string]any)
	for i := 0; i < 100; i++ {
		payload[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value %d", i)
	}

	params := lingo.LocalizationParams{TargetLocale: "es"}
	_, err = client.LocalizeObject(context.Background(), payload, params, false)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	count := callCount.Load()
	if count <= 1 {
		t.Fatalf("expected more than 1 chunk, got %d server calls", count)
	}
	t.Logf("Number of chunks: %d", count)
}

func TestMocked_ReferenceParameterIncluded(t *testing.T) {
	var capturedBody map[string]any
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		capturedBody = body
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"key": "translated"},
		})
	}))
	defer server.Close()

	client, err := lingo.NewClient("test-key", lingo.SetURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	ref := map[string]map[string]any{
		"es": {"key": "valor de referencia"},
	}
	obj := map[string]any{"key": "value"}
	params := lingo.LocalizationParams{
		TargetLocale: "es",
		Reference:    ref,
	}
	_, err = client.LocalizeObject(context.Background(), obj, params, false)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if capturedBody == nil {
		t.Fatal("no request body captured")
	}
	refField, ok := capturedBody["reference"]
	if !ok {
		t.Fatal("expected 'reference' field in request body")
	}
	refMap, ok := refField.(map[string]any)
	if !ok {
		t.Fatalf("expected reference to be map, got %T", refField)
	}
	esRef, ok := refMap["es"]
	if !ok {
		t.Fatal("expected 'es' key in reference")
	}
	esRefMap, ok := esRef.(map[string]any)
	if !ok {
		t.Fatalf("expected es reference to be map, got %T", esRef)
	}
	if esRefMap["key"] != "valor de referencia" {
		t.Errorf("expected reference value 'valor de referencia', got %v", esRefMap["key"])
	}
}

func TestMocked_WorkflowIDConsistency(t *testing.T) {
	var workflowIDs []string
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		params, ok := body["params"].(map[string]any)
		if ok {
			wfID, ok := params["workflowId"].(string)
			if ok {
				mu.Lock()
				workflowIDs = append(workflowIDs, wfID)
				mu.Unlock()
			}
		}

		data, ok := body["data"].(map[string]any)
		if !ok {
			data = map[string]any{}
		}
		response := make(map[string]any)
		for k := range data {
			response[k] = "translated"
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": response})
	}))
	defer server.Close()

	client, err := lingo.NewClient("test-key", lingo.SetURL(server.URL), lingo.SetBatchSize(2))
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	payload := make(map[string]any)
	for i := 0; i < 50; i++ {
		payload[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value %d", i)
	}

	params := lingo.LocalizationParams{TargetLocale: "es"}
	_, err = client.LocalizeObject(context.Background(), payload, params, false)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(workflowIDs) < 2 {
		t.Fatalf("expected at least 2 workflow IDs (chunks), got %d", len(workflowIDs))
	}
	firstID := workflowIDs[0]
	if firstID == "" {
		t.Fatal("expected non-empty workflow ID")
	}
	for i, id := range workflowIDs {
		if id != firstID {
			t.Errorf("workflow ID mismatch: chunk 0 has %q, chunk %d has %q", firstID, i, id)
		}
	}
	t.Logf("All %d chunks used workflow ID: %s", len(workflowIDs), firstID)
}

func TestMocked_ConcurrentChunkProcessing(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		time.Sleep(50 * time.Millisecond)

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		data, ok := body["data"].(map[string]any)
		if !ok {
			data = map[string]any{}
		}
		response := make(map[string]any)
		for k := range data {
			response[k] = "translated"
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": response})
	}))
	defer server.Close()

	client, err := lingo.NewClient("test-key", lingo.SetURL(server.URL), lingo.SetBatchSize(2))
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	payload := make(map[string]any)
	for i := 0; i < 10; i++ {
		payload[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value %d", i)
	}

	params := lingo.LocalizationParams{TargetLocale: "es"}

	start := time.Now()
	_, err = client.LocalizeObject(context.Background(), payload, params, true)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	duration := time.Since(start)

	count := int(callCount.Load())
	serialDuration := time.Duration(count) * 50 * time.Millisecond
	if duration >= serialDuration {
		t.Errorf("concurrent processing (%s) was not faster than serial estimate (%s with %d calls)", duration, serialDuration, count)
	}
	t.Logf("Concurrent duration: %s, call count: %d, serial estimate: %s", duration, count, serialDuration)
}

func TestMocked_RetryOn500(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := callCount.Add(1)
		if count <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "server error"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"text": "translated"},
		})
	}))
	defer server.Close()

	client, err := lingo.NewClient("test-key", lingo.SetURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	params := lingo.LocalizationParams{TargetLocale: "es"}
	result, err := client.LocalizeText(context.Background(), "hello", params)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if result != "translated" {
		t.Errorf("expected 'translated', got '%s'", result)
	}

	finalCount := callCount.Load()
	if finalCount != 3 {
		t.Errorf("expected 3 server calls (2 retries + 1 success), got %d", finalCount)
	}
}

func TestMocked_NoRetryOn400(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "bad request"}`))
	}))
	defer server.Close()

	client, err := lingo.NewClient("test-key", lingo.SetURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	params := lingo.LocalizationParams{TargetLocale: "es"}
	_, err = client.LocalizeText(context.Background(), "hello", params)
	if err == nil {
		t.Fatal("expected error for bad request")
	}
	var ve *lingo.ValueError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValueError, got %T: %s", err, err)
	}

	finalCount := callCount.Load()
	if finalCount != 1 {
		t.Errorf("expected exactly 1 server call (no retries), got %d", finalCount)
	}
}

func TestMocked_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"text": "translated"},
		})
	}))
	defer server.Close()

	client, err := lingo.NewClient("test-key", lingo.SetURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	params := lingo.LocalizationParams{TargetLocale: "es"}
	_, err = client.LocalizeText(ctx, "hello", params)
	if err == nil {
		t.Fatal("expected error for context cancellation")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") && !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("expected context error, got: %s", err)
	}
}
