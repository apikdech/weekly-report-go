package llm

import "testing"

func TestParseProvider(t *testing.T) {
	tests := []struct {
		in   string
		want ProviderType
	}{
		{"", Gemini},
		{"   ", Gemini},
		{"gemini", Gemini},
		{"GEMINI", Gemini},
		{"openai", OpenAI},
		{"anthropic", Anthropic},
		{"unknown", Gemini},
	}
	for _, tt := range tests {
		got := ParseProvider(tt.in)
		if got != tt.want {
			t.Errorf("ParseProvider(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
