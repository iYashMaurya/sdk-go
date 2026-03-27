package tests

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	lingo "github.com/lingodotdev/sdk-go"
)

func TestNewClient_MissingAPIKey(t *testing.T) {
	_, err := lingo.NewClient("")
	if err == nil {
		t.Fatal("expected error for missing api key")
	}
	var ve *lingo.ValueError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValueError, got %T", err)
	}
	if !strings.Contains(ve.Message, "api key") {
		t.Errorf("expected error about api key, got: %s", ve.Message)
	}
}

func TestNewClient_InvalidURL(t *testing.T) {
	_, err := lingo.NewClient("test-key", lingo.SetURL("not-a-url"))
	if err == nil {
		t.Fatal("expected error for invalid url")
	}
	var ve *lingo.ValueError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValueError, got %T", err)
	}
	if !strings.Contains(ve.Message, "url") {
		t.Errorf("expected error about url, got: %s", ve.Message)
	}
}

func TestNewClient_InvalidBatchSize(t *testing.T) {
	_, err := lingo.NewClient("test-key", lingo.SetBatchSize(0))
	if err == nil {
		t.Fatal("expected error for batch size 0")
	}
	var ve *lingo.ValueError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValueError, got %T", err)
	}
	if !strings.Contains(ve.Message, "batch size") {
		t.Errorf("expected error about batch size, got: %s", ve.Message)
	}

	_, err = lingo.NewClient("test-key", lingo.SetBatchSize(300))
	if err == nil {
		t.Fatal("expected error for batch size > 250")
	}
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValueError, got %T", err)
	}
}

func TestNewClient_InvalidIdealBatchItemSize(t *testing.T) {
	_, err := lingo.NewClient("test-key", lingo.SetIdealBatchItemSize(0))
	if err == nil {
		t.Fatal("expected error for ideal batch item size 0")
	}
	var ve *lingo.ValueError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValueError, got %T", err)
	}
	if !strings.Contains(ve.Message, "ideal batch item size") {
		t.Errorf("expected error about ideal batch item size, got: %s", ve.Message)
	}

	_, err = lingo.NewClient("test-key", lingo.SetIdealBatchItemSize(3000))
	if err == nil {
		t.Fatal("expected error for ideal batch item size > 2500")
	}
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValueError, got %T", err)
	}
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		name     string
		payload  any
		expected int
	}{
		{name: "empty string", payload: "", expected: 0},
		{name: "single word", payload: "hello", expected: 1},
		{name: "multiple words", payload: "hello world foo", expected: 3},
		{name: "nested map", payload: map[string]any{"a": "one two", "b": "three"}, expected: 3},
		{name: "nested slice", payload: []any{"one two", "three four five"}, expected: 5},
		{name: "integer", payload: 42, expected: 0},
		{name: "nil", payload: nil, expected: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lingo.CountWords(tt.payload)
			if result != tt.expected {
				t.Errorf("CountWords(%v) = %d, want %d", tt.payload, result, tt.expected)
			}
		})
	}
}

func TestExtractChunks_RespectsItemLimit(t *testing.T) {
	client, err := lingo.NewClient("test-key", lingo.SetBatchSize(2))
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	payload := map[string]any{
		"a": "hello",
		"b": "world",
		"c": "foo",
		"d": "bar",
	}

	chunks := client.ExtractChunks(payload)
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks with batch size 2 and 4 items, got %d", len(chunks))
	}

	for i, chunk := range chunks {
		if len(chunk) > 2 {
			t.Errorf("chunk %d has %d items, expected at most 2", i, len(chunk))
		}
	}
}

func TestExtractChunks_PreservesAllKeys(t *testing.T) {
	client, err := lingo.NewClient("test-key", lingo.SetBatchSize(2))
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	payload := map[string]any{
		"a": "hello",
		"b": "world",
		"c": "foo",
		"d": "bar",
	}

	chunks := client.ExtractChunks(payload)
	allKeys := make(map[string]bool)
	for _, chunk := range chunks {
		for k := range chunk {
			allKeys[k] = true
		}
	}
	if len(allKeys) != len(payload) {
		t.Errorf("expected %d total keys across all chunks, got %d", len(payload), len(allKeys))
	}
	for k := range payload {
		if !allKeys[k] {
			t.Errorf("key %q missing from chunks", k)
		}
	}
}

func TestExtractChunks_EmptyPayload(t *testing.T) {
	client, err := lingo.NewClient("test-key")
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	chunks := client.ExtractChunks(map[string]any{})
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty payload, got %d", len(chunks))
	}
}

func TestLocalizeText_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Authorization 'Bearer test-key', got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json; charset=utf-8" {
			t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
		}
		if !strings.HasSuffix(r.URL.Path, "/i18n") {
			t.Errorf("expected request path ending with /i18n, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %s", err)
		}
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("failed to decode request body: %s", err)
		}

		localeMap, ok := reqBody["locale"].(map[string]any)
		if !ok {
			t.Fatal("expected locale field in request body")
		}
		if localeMap["target"] != "es" {
			t.Errorf("expected locale.target 'es', got %v", localeMap["target"])
		}

		dataMap, ok := reqBody["data"].(map[string]any)
		if !ok {
			t.Fatal("expected data field in request body")
		}
		if dataMap["text"] != "hello" {
			t.Errorf("expected data.text 'hello', got %v", dataMap["text"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"text": "hola"},
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
	if result != "hola" {
		t.Fatalf("expected 'hola', got '%s'", result)
	}
}

func TestLocalizeText_EmptyText(t *testing.T) {
	client, err := lingo.NewClient("test-key")
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	params := lingo.LocalizationParams{TargetLocale: "es"}
	_, err = client.LocalizeText(context.Background(), "", params)
	if err == nil {
		t.Fatal("expected error for empty text")
	}
	var ve *lingo.ValueError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValueError, got %T", err)
	}
	if !strings.Contains(ve.Message, "text must not be empty") {
		t.Errorf("expected error about empty text, got: %s", ve.Message)
	}
}

func TestLocalizeText_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	client, err := lingo.NewClient("test-key", lingo.SetURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	params := lingo.LocalizationParams{TargetLocale: "es"}
	_, err = client.LocalizeText(context.Background(), "hello", params)
	if err == nil {
		t.Fatal("expected error for server error")
	}
	var re *lingo.RuntimeError
	if !errors.As(err, &re) {
		t.Fatalf("expected *RuntimeError, got %T", err)
	}
	if !strings.Contains(re.Message, "server error") {
		t.Errorf("expected error message to contain 'server error', got: %s", re.Message)
	}
}

func TestLocalizeText_BadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	if !strings.Contains(ve.Message, "invalid request") {
		t.Errorf("expected error about invalid request, got: %s", ve.Message)
	}
}

func TestLocalizeText_StreamingError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"error": "streaming error occurred",
		})
	}))
	defer server.Close()

	client, err := lingo.NewClient("test-key", lingo.SetURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	params := lingo.LocalizationParams{TargetLocale: "es"}
	_, err = client.LocalizeText(context.Background(), "hello", params)
	if err == nil {
		t.Fatal("expected error for streaming error response")
	}
	var re *lingo.RuntimeError
	if !errors.As(err, &re) {
		t.Fatalf("expected *RuntimeError, got %T", err)
	}
	if !strings.Contains(re.Message, "streaming error occurred") {
		t.Errorf("expected error about streaming error, got: %s", re.Message)
	}
}

func TestLocalizeChat_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"chat": []any{
					map[string]any{"name": "user", "text": "hola"},
					map[string]any{"name": "bot", "text": "adiós"},
				},
			},
		})
	}))
	defer server.Close()

	client, err := lingo.NewClient("test-key", lingo.SetURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	chat := []map[string]string{
		{"name": "user", "text": "hello"},
		{"name": "bot", "text": "goodbye"},
	}
	params := lingo.LocalizationParams{TargetLocale: "es"}
	result, err := client.LocalizeChat(context.Background(), chat, params)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	if result[0]["name"] != "user" || result[0]["text"] != "hola" {
		t.Errorf("unexpected first message: %v", result[0])
	}
	if result[1]["name"] != "bot" || result[1]["text"] != "adiós" {
		t.Errorf("unexpected second message: %v", result[1])
	}
}

func TestLocalizeChat_MissingNameField(t *testing.T) {
	client, err := lingo.NewClient("test-key")
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	chat := []map[string]string{
		{"text": "hello"},
	}
	params := lingo.LocalizationParams{TargetLocale: "es"}
	_, err = client.LocalizeChat(context.Background(), chat, params)
	if err == nil {
		t.Fatal("expected error for missing name field")
	}
	var ve *lingo.ValueError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValueError, got %T", err)
	}
	if !strings.Contains(ve.Message, "index 0") {
		t.Errorf("expected error mentioning index 0, got: %s", ve.Message)
	}
	if !strings.Contains(ve.Message, "name") {
		t.Errorf("expected error mentioning 'name', got: %s", ve.Message)
	}
}

func TestLocalizeChat_MissingTextField(t *testing.T) {
	client, err := lingo.NewClient("test-key")
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	chat := []map[string]string{
		{"name": "user"},
	}
	params := lingo.LocalizationParams{TargetLocale: "es"}
	_, err = client.LocalizeChat(context.Background(), chat, params)
	if err == nil {
		t.Fatal("expected error for missing text field")
	}
	var ve *lingo.ValueError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValueError, got %T", err)
	}
	if !strings.Contains(ve.Message, "index 0") {
		t.Errorf("expected error mentioning index 0, got: %s", ve.Message)
	}
	if !strings.Contains(ve.Message, "text") {
		t.Errorf("expected error mentioning 'text', got: %s", ve.Message)
	}
}

func TestLocalizeChat_EmptyChat(t *testing.T) {
	client, err := lingo.NewClient("test-key")
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	params := lingo.LocalizationParams{TargetLocale: "es"}
	result, err := client.LocalizeChat(context.Background(), []map[string]string{}, params)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty result, got %d items", len(result))
	}
}

func TestLocalizeChat_ResponseLengthMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"chat": []any{
					map[string]any{"name": "user", "text": "hola"},
				},
			},
		})
	}))
	defer server.Close()

	client, err := lingo.NewClient("test-key", lingo.SetURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	chat := []map[string]string{
		{"name": "user", "text": "hello"},
		{"name": "bot", "text": "goodbye"},
	}
	params := lingo.LocalizationParams{TargetLocale: "es"}
	_, err = client.LocalizeChat(context.Background(), chat, params)
	if err == nil {
		t.Fatal("expected error for response length mismatch")
	}
	var re *lingo.RuntimeError
	if !errors.As(err, &re) {
		t.Fatalf("expected *RuntimeError, got %T", err)
	}
	if !strings.Contains(re.Message, "expected 2") {
		t.Errorf("expected error about message count mismatch, got: %s", re.Message)
	}
}

func TestBatchLocalizeText_EmptyLocales(t *testing.T) {
	client, err := lingo.NewClient("test-key")
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	result, err := client.BatchLocalizeText(context.Background(), "hello", nil, nil, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected empty result, got %d items", len(result))
	}
}

func TestBatchLocalizeText_Success(t *testing.T) {
	translations := map[string]string{
		"es": "hola",
		"fr": "bonjour",
		"de": "hallo",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %s", err)
			return
		}
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Errorf("failed to decode request body: %s", err)
			return
		}

		localeMap, ok := reqBody["locale"].(map[string]any)
		if !ok {
			t.Error("expected locale field in request body")
			return
		}
		target, ok := localeMap["target"].(string)
		if !ok {
			t.Error("expected locale.target string in request body")
			return
		}

		translated, exists := translations[target]
		if !exists {
			translated = "unknown"
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"text": translated},
		})
	}))
	defer server.Close()

	client, err := lingo.NewClient("test-key", lingo.SetURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	locales := []string{"es", "fr", "de"}
	results, err := client.BatchLocalizeText(context.Background(), "hello", nil, nil, locales)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for i, locale := range locales {
		expected := translations[locale]
		if results[i] != expected {
			t.Errorf("locale %s: expected '%s', got '%s'", locale, expected, results[i])
		}
	}
}

func TestBatchLocalizeText_EmptyText(t *testing.T) {
	client, err := lingo.NewClient("test-key")
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	_, err = client.BatchLocalizeText(context.Background(), "", nil, nil, []string{"es"})
	if err == nil {
		t.Fatal("expected error for empty text")
	}
	var ve *lingo.ValueError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValueError, got %T", err)
	}
	if !strings.Contains(ve.Message, "text must not be empty") {
		t.Errorf("expected error about empty text, got: %s", ve.Message)
	}
}

// --- WHOAMI TESTS ---

func TestWhoAmI_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"email": "test@example.com",
				"id":    "user-123",
			},
		})
	}))
	defer server.Close()

	client, err := lingo.NewClient("test-key", lingo.SetURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	result, err := client.WhoAmI(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result["email"] != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%s'", result["email"])
	}
	if result["id"] != "user-123" {
		t.Errorf("expected id 'user-123', got '%s'", result["id"])
	}
}

func TestWhoAmI_Unauthenticated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "unauthorized"}`))
	}))
	defer server.Close()

	client, err := lingo.NewClient("test-key", lingo.SetURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	result, err := client.WhoAmI(context.Background())
	if err != nil {
		t.Fatalf("expected nil error for unauthenticated, got: %s", err)
	}
	if result != nil {
		t.Fatalf("expected nil result for unauthenticated, got: %v", result)
	}
}

func TestWhoAmI_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	client, err := lingo.NewClient("test-key", lingo.SetURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	_, err = client.WhoAmI(context.Background())
	if err == nil {
		t.Fatal("expected error for server error")
	}
	var re *lingo.RuntimeError
	if !errors.As(err, &re) {
		t.Fatalf("expected *RuntimeError, got %T", err)
	}
	if !strings.Contains(re.Message, "server error") {
		t.Errorf("expected error about server error, got: %s", re.Message)
	}
}

func TestRecognizeLocale_EmptyText(t *testing.T) {
	client, err := lingo.NewClient("test-key")
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	_, err = client.RecognizeLocale(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty text")
	}
	var ve *lingo.ValueError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValueError, got %T", err)
	}
	if !strings.Contains(ve.Message, "text must not be empty") {
		t.Errorf("expected error about empty text, got: %s", ve.Message)
	}
}

func TestRecognizeLocale_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/recognize") {
			t.Errorf("expected path ending with /recognize, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %s", err)
		}
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("failed to decode request body: %s", err)
		}
		if reqBody["text"] != "hello world" {
			t.Errorf("expected text 'hello world', got %v", reqBody["text"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"locale": "en",
		})
	}))
	defer server.Close()

	client, err := lingo.NewClient("test-key", lingo.SetURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	locale, err := client.RecognizeLocale(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if locale != "en" {
		t.Fatalf("expected 'en', got '%s'", locale)
	}
}

func TestRecognizeLocale_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	client, err := lingo.NewClient("test-key", lingo.SetURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	_, err = client.RecognizeLocale(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error for server error")
	}
	var re *lingo.RuntimeError
	if !errors.As(err, &re) {
		t.Fatalf("expected *RuntimeError, got %T", err)
	}
	if !strings.Contains(re.Message, "server error") {
		t.Errorf("expected error about server error, got: %s", re.Message)
	}
}

func TestTruncateResponse_Short(t *testing.T) {
	short := "hello"
	result := lingo.TruncateResponse(short)
	if result != short {
		t.Errorf("expected '%s', got '%s'", short, result)
	}
}

func TestTruncateResponse_Long(t *testing.T) {
	long := strings.Repeat("x", 300)
	result := lingo.TruncateResponse(long)
	if len(result) != lingo.MaxResponseLength+3 {
		t.Errorf("expected length %d, got %d", lingo.MaxResponseLength+3, len(result))
	}
	if !strings.HasSuffix(result, "...") {
		t.Errorf("expected truncated string to end with '...', got '%s'", result[len(result)-3:])
	}
}

func TestTruncateResponse_ExactLength(t *testing.T) {
	exact := strings.Repeat("x", lingo.MaxResponseLength)
	result := lingo.TruncateResponse(exact)
	if result != exact {
		t.Errorf("expected string of length %d to not be truncated", lingo.MaxResponseLength)
	}
}
