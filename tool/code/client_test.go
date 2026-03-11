package code

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TencentCloudAgentRuntime/ags-go-sdk/connection"
)

func TestRunCode_MaxBufferSize(t *testing.T) {
	// Generate a long string (70KB, larger than default 64KB initial buffer)
	longData := strings.Repeat("a", 70*1024)
	respData := map[string]any{
		"type": "stdout",
		"text": longData,
	}
	jsonBytes, err := json.Marshal(respData)
	if err != nil {
		panic(err)
	}
	jsonLine := string(jsonBytes) + "\n"

	// Create a TLS server to match the client's HTTPS requirement
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, jsonLine)
	}))
	defer server.Close()

	// Extract host from server URL (remove https:// prefix)
	serverURL := server.URL
	domain := strings.TrimPrefix(serverURL, "https://")

	cfg := &connection.Config{
		Domain: domain,
	}
	client := New(cfg)
	// Inject the server's client which trusts the test server's certificate
	client.httpClient = server.Client()

	t.Run("SufficientBuffer", func(t *testing.T) {
		// Set buffer to 100KB (larger than the ~70KB line)
		runConfig := &RunCodeConfig{
			MaxBufferSize: 100 * 1024,
		}
		exec, err := client.RunCode(context.Background(), "print('hello')", runConfig, nil)
		if err != nil {
			t.Fatalf("Expected success, got error: %v", err)
		}
		if len(exec.Logs.Stdout) != 1 || exec.Logs.Stdout[0] != longData {
			t.Errorf("Unexpected stdout content")
		}
	})

	t.Run("InsufficientBuffer", func(t *testing.T) {
		// Set buffer to 65KB (larger than initial 64KB, but smaller than 70KB data)
		// Note: Since initial buffer is 64KB, we must test with data > 64KB to verify limit
		runConfig := &RunCodeConfig{
			MaxBufferSize: 65 * 1024,
		}
		_, err := client.RunCode(context.Background(), "print('hello')", runConfig, nil)
		if err == nil {
			t.Fatal("Expected error due to small buffer, got nil")
		}
		// Expect bufio.Scanner: token too long
		if !strings.Contains(err.Error(), "token too long") {
			t.Errorf("Expected 'token too long' error, got: %v", err)
		}
	})

	t.Run("DefaultBuffer", func(t *testing.T) {
		// Default buffer is 1GB, should handle 70KB easily
		runConfig := &RunCodeConfig{}
		exec, err := client.RunCode(context.Background(), "print('hello')", runConfig, nil)
		if err != nil {
			t.Fatalf("Expected success with default buffer, got error: %v", err)
		}
		if len(exec.Logs.Stdout) != 1 || exec.Logs.Stdout[0] != longData {
			t.Errorf("Unexpected stdout content")
		}
	})
}

func TestRunCode_ContextCancellation(t *testing.T) {
	t.Run("ContextTimeoutDuringResponse", func(t *testing.T) {
		// Create a server that sends initial data then hangs
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			
			// Send initial stdout message
			initialMsg := map[string]any{
				"type": "stdout",
				"text": "Starting execution...",
			}
			jsonBytes, _ := json.Marshal(initialMsg)
			fmt.Fprintf(w, "%s\n", jsonBytes)
			
			// Flush to ensure data is sent
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			
			// Simulate a long-running process that never sends more data
			// This tests the scenario where remote doesn't respond
			time.Sleep(5 * time.Second)
			
			// Send final message (this should not be reached due to context timeout)
			finalMsg := map[string]any{
				"type": "stdout", 
				"text": "This should not be received",
			}
			jsonBytes, _ = json.Marshal(finalMsg)
			fmt.Fprintf(w, "%s\n", jsonBytes)
		}))
		defer server.Close()

		// Extract domain from server URL
		domain := strings.TrimPrefix(server.URL, "https://")

		cfg := &connection.Config{
			Domain: domain,
		}
		client := New(cfg)
		client.httpClient = server.Client()

		// Create context with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		start := time.Now()
		exec, err := client.RunCode(ctx, "print('hello')", &RunCodeConfig{}, nil)
		duration := time.Since(start)

		// Should return context deadline exceeded error
		if err == nil {
			t.Fatal("Expected context deadline exceeded error, got nil")
		}
		
		if !strings.Contains(err.Error(), "context deadline exceeded") {
			t.Errorf("Expected 'context deadline exceeded' error, got: %v", err)
		}

		// Should return quickly (within timeout + small buffer)
		if duration > 2*time.Second {
			t.Errorf("Expected quick return due to context timeout, took %v", duration)
		}

		// Should have received the initial message before timeout
		if exec != nil && len(exec.Logs.Stdout) > 0 {
			if exec.Logs.Stdout[0] != "Starting execution..." {
				t.Errorf("Expected initial stdout message, got: %v", exec.Logs.Stdout[0])
			}
		}
	})

	t.Run("ContextCancelledBeforeRequest", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Server handler should not be called when context is already cancelled")
		}))
		defer server.Close()

		domain := strings.TrimPrefix(server.URL, "https://")
		cfg := &connection.Config{
			Domain: domain,
		}
		client := New(cfg)
		client.httpClient = server.Client()

		// Cancel context before making request
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := client.RunCode(ctx, "print('hello')", &RunCodeConfig{}, nil)
		
		if err == nil {
			t.Fatal("Expected context cancelled error, got nil")
		}
		
		if !strings.Contains(err.Error(), "context canceled") {
			t.Errorf("Expected 'context canceled' error, got: %v", err)
		}
	})

	t.Run("ContextTimeoutDuringInitialConnection", func(t *testing.T) {
		// Create a server that delays the initial response
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Delay before sending any response
			time.Sleep(2 * time.Second)
			
			w.Header().Set("Content-Type", "application/json")
			msg := map[string]any{
				"type": "stdout",
				"text": "Should not reach here",
			}
			jsonBytes, _ := json.Marshal(msg)
			fmt.Fprintf(w, "%s\n", jsonBytes)
		}))
		defer server.Close()

		domain := strings.TrimPrefix(server.URL, "https://")
		cfg := &connection.Config{
			Domain: domain,
		}
		client := New(cfg)
		client.httpClient = server.Client()

		// Short timeout that should trigger before server responds
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		start := time.Now()
		_, err := client.RunCode(ctx, "print('hello')", &RunCodeConfig{}, nil)
		duration := time.Since(start)

		if err == nil {
			t.Fatal("Expected context deadline exceeded error, got nil")
		}

		if !strings.Contains(err.Error(), "context deadline exceeded") {
			t.Errorf("Expected 'context deadline exceeded' error, got: %v", err)
		}

		// Should timeout quickly
		if duration > 1*time.Second {
			t.Errorf("Expected quick timeout, took %v", duration)
		}
	})
}

func TestRunCode_ServerConnectionAwareness(t *testing.T) {
	t.Run("ServerDetectsClientDisconnection", func(t *testing.T) {
		// Channel to track server-side events
		serverEvents := make(chan string, 10)
		clientDisconnected := make(chan bool, 1)
		
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			serverEvents <- "request_received"
			
			w.Header().Set("Content-Type", "application/json")
			
			// Send initial message
			initialMsg := map[string]any{
				"type": "stdout",
				"text": "Server started processing...",
			}
			jsonBytes, _ := json.Marshal(initialMsg)
			fmt.Fprintf(w, "%s\n", jsonBytes)
			
			// Flush to ensure data is sent
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			serverEvents <- "initial_data_sent"
			
			// Simulate server continuing to work and trying to send more data
			for i := 0; i < 10; i++ {
				time.Sleep(200 * time.Millisecond)
				
				// Try to send more data
				msg := map[string]any{
					"type": "stdout",
					"text": fmt.Sprintf("Processing step %d...", i+1),
				}
				jsonBytes, _ := json.Marshal(msg)
				
				// Write data
				n, err := fmt.Fprintf(w, "%s\n", jsonBytes)
				if err != nil {
					serverEvents <- fmt.Sprintf("write_error: %v", err)
					clientDisconnected <- true
					return
				}
				
				// Try to flush
				if flusher, ok := w.(http.Flusher); ok {
					// In HTTP/1.1, Flush() might detect client disconnection
					flusher.Flush()
				}
				
				// Check if we can detect disconnection through write size
				if n == 0 {
					serverEvents <- "zero_bytes_written"
					clientDisconnected <- true
					return
				}
				
				serverEvents <- fmt.Sprintf("step_%d_sent", i+1)
			}
			
			serverEvents <- "server_completed_normally"
		}))
		defer server.Close()

		domain := strings.TrimPrefix(server.URL, "https://")
		cfg := &connection.Config{
			Domain: domain,
		}
		client := New(cfg)
		client.httpClient = server.Client()

		// Create context with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
		defer cancel()

		start := time.Now()
		exec, err := client.RunCode(ctx, "print('hello')", &RunCodeConfig{}, nil)
		duration := time.Since(start)

		// Client should timeout
		if err == nil {
			t.Fatal("Expected context deadline exceeded error, got nil")
		}
		
		if !strings.Contains(err.Error(), "context deadline exceeded") {
			t.Errorf("Expected 'context deadline exceeded' error, got: %v", err)
		}

		// Should return quickly due to timeout
		if duration > 1200*time.Millisecond {
			t.Errorf("Expected quick return due to context timeout, took %v", duration)
		}

		// Should have received initial data
		if exec != nil && len(exec.Logs.Stdout) > 0 {
			if exec.Logs.Stdout[0] != "Server started processing..." {
				t.Errorf("Expected initial stdout message, got: %v", exec.Logs.Stdout[0])
			}
		}

		// Wait a bit more to see if server detects disconnection
		select {
		case <-clientDisconnected:
			t.Log("✅ Server successfully detected client disconnection")
		case <-time.After(2 * time.Second):
			t.Log("⚠️  Server did not detect client disconnection within timeout")
			// This is not necessarily a failure, as detection depends on OS and HTTP implementation
		}

		// Collect server events for analysis
		close(serverEvents)
		var events []string
		for event := range serverEvents {
			events = append(events, event)
		}
		
		t.Logf("Server events: %v", events)
		
		// Verify server received request and sent initial data
		hasRequestReceived := false
		hasInitialDataSent := false
		for _, event := range events {
			if event == "request_received" {
				hasRequestReceived = true
			}
			if event == "initial_data_sent" {
				hasInitialDataSent = true
			}
		}
		
		if !hasRequestReceived {
			t.Error("Server should have received the request")
		}
		if !hasInitialDataSent {
			t.Error("Server should have sent initial data")
		}
	})

	t.Run("ServerResourceCleanupOnClientDisconnection", func(t *testing.T) {
		// Track server-side resource usage
		serverStarted := make(chan bool, 1)
		serverFinished := make(chan string, 1)
		
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			serverStarted <- true
			defer func() {
				if r := recover(); r != nil {
					serverFinished <- fmt.Sprintf("panic: %v", r)
				} else {
					serverFinished <- "normal_exit"
				}
			}()
			
			w.Header().Set("Content-Type", "application/json")
			
			// Send initial response
			msg := map[string]any{
				"type": "stdout",
				"text": "Starting long operation...",
			}
			jsonBytes, _ := json.Marshal(msg)
			fmt.Fprintf(w, "%s\n", jsonBytes)
			
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			
			// Simulate long-running operation
			for i := 0; i < 50; i++ {
				time.Sleep(100 * time.Millisecond)
				
				// Try to write data - this should eventually fail when client disconnects
				msg := map[string]any{
					"type": "stdout", 
					"text": fmt.Sprintf("Long operation step %d", i),
				}
				jsonBytes, _ := json.Marshal(msg)
				
				_, err := fmt.Fprintf(w, "%s\n", jsonBytes)
				if err != nil {
					// Client disconnected
					serverFinished <- fmt.Sprintf("client_disconnected_at_step_%d", i)
					return
				}
				
				// Flush to trigger potential connection errors
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
			}
			
			serverFinished <- "completed_all_steps"
		}))
		defer server.Close()

		domain := strings.TrimPrefix(server.URL, "https://")
		cfg := &connection.Config{
			Domain: domain,
		}
		client := New(cfg)
		client.httpClient = server.Client()

		// Very short timeout to force early disconnection
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()

		// Wait for server to start
		go func() {
			client.RunCode(ctx, "print('test')", &RunCodeConfig{}, nil)
		}()

		// Wait for server to start processing
		select {
		case <-serverStarted:
			t.Log("Server started processing request")
		case <-time.After(1 * time.Second):
			t.Fatal("Server did not start processing within timeout")
		}

		// Wait for server to finish (either normally or due to client disconnection)
		select {
		case result := <-serverFinished:
			t.Logf("Server finished with: %s", result)
			
			// Ideally, server should detect client disconnection
			if strings.Contains(result, "client_disconnected") {
				t.Log("✅ Server detected client disconnection and cleaned up resources")
			} else if result == "completed_all_steps" {
				t.Log("⚠️  Server completed all steps without detecting disconnection")
				// This might happen depending on OS buffering and HTTP implementation
			} else {
				t.Logf("Server finished with unexpected result: %s", result)
			}
			
		case <-time.After(10 * time.Second):
			t.Error("Server did not finish within reasonable time - possible resource leak")
		}
	})
}

func TestRunCode_NetworkBehaviorOnTimeout(t *testing.T) {
	t.Run("AnalyzeConnectionCloseSequence", func(t *testing.T) {
		connectionClosed := make(chan bool, 1)
		serverWriteAttempts := make(chan int, 1)
		
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Monitor connection state
			defer func() {
				connectionClosed <- true
			}()
			
			w.Header().Set("Content-Type", "application/json")
			
			// Send initial data
			msg := map[string]any{
				"type": "stdout",
				"text": "Initial message",
			}
			jsonBytes, _ := json.Marshal(msg)
			fmt.Fprintf(w, "%s\n", jsonBytes)
			
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			
			// Try to write more data and count attempts until failure
			writeCount := 0
			for i := 0; i < 20; i++ {
				time.Sleep(100 * time.Millisecond)
				writeCount++
				
				msg := map[string]any{
					"type": "stdout",
					"text": fmt.Sprintf("Message %d", i),
				}
				jsonBytes, _ := json.Marshal(msg)
				
				_, err := fmt.Fprintf(w, "%s\n", jsonBytes)
				if err != nil {
					t.Logf("Server write failed at attempt %d: %v", writeCount, err)
					serverWriteAttempts <- writeCount
					return
				}
				
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
			}
			
			serverWriteAttempts <- writeCount
		}))
		defer server.Close()

		domain := strings.TrimPrefix(server.URL, "https://")
		cfg := &connection.Config{
			Domain: domain,
		}
		client := New(cfg)
		client.httpClient = server.Client()

		// Short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		start := time.Now()
		_, err := client.RunCode(ctx, "test", &RunCodeConfig{}, nil)
		clientDuration := time.Since(start)

		// Verify client behavior
		if err == nil {
			t.Fatal("Expected timeout error")
		}
		
		if !strings.Contains(err.Error(), "context deadline exceeded") {
			t.Errorf("Expected context deadline exceeded, got: %v", err)
		}

		t.Logf("Client returned after: %v", clientDuration)

		// Wait for server to detect disconnection
		select {
		case writeAttempts := <-serverWriteAttempts:
			t.Logf("Server made %d write attempts before detecting disconnection", writeAttempts)
		case <-time.After(3 * time.Second):
			t.Error("Server did not detect disconnection within timeout")
		}

		// Verify server handler finished
		select {
		case <-connectionClosed:
			t.Log("Server handler finished")
		case <-time.After(1 * time.Second):
			t.Error("Server handler did not finish")
		}
	})
}
