package connection

import (
	"testing"
)

func TestGetScheme_Default(t *testing.T) {
	config := &Config{}
	if got := config.GetScheme(); got != DefaultScheme {
		t.Errorf("expected default scheme '%s', got '%s'", DefaultScheme, got)
	}
}

func TestGetScheme_HTTPS(t *testing.T) {
	config := &Config{Scheme: "https"}
	if got := config.GetScheme(); got != "https" {
		t.Errorf("expected scheme 'https', got '%s'", got)
	}
}

func TestGetScheme_HTTP(t *testing.T) {
	config := &Config{Scheme: "http"}
	if got := config.GetScheme(); got != "http" {
		t.Errorf("expected scheme 'http', got '%s'", got)
	}
}

func TestGetScheme_EmptyStringReturnsDefault(t *testing.T) {
	config := &Config{Scheme: ""}
	if got := config.GetScheme(); got != DefaultScheme {
		t.Errorf("expected default scheme '%s' for empty string, got '%s'", DefaultScheme, got)
	}
}

func TestDefaultScheme_Constant(t *testing.T) {
	if DefaultScheme != "https" {
		t.Errorf("expected DefaultScheme to be 'https', got '%s'", DefaultScheme)
	}
}
