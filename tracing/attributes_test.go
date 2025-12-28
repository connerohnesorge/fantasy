package tracing

import (
	"strings"
	"testing"
)

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxBytes int
		wantLen  int
		wantEnd  string
	}{
		{
			name:     "short string no truncation",
			input:    "hello",
			maxBytes: 100,
			wantLen:  5,
			wantEnd:  "hello",
		},
		{
			name:     "long string truncated",
			input:    strings.Repeat("a", 200),
			maxBytes: 50,
			wantLen:  50,
			wantEnd:  "...",
		},
		{
			name:     "utf8 string truncated at boundary",
			input:    "Hello 世界世界世界世界世界",
			maxBytes: 20,
			wantLen:  20,
			wantEnd:  "...",
		},
		{
			name:     "empty string",
			input:    "",
			maxBytes: 10,
			wantLen:  0,
			wantEnd:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncateString(tt.input, tt.maxBytes)
			if len(got) > tt.maxBytes {
				t.Errorf("TruncateString() length = %d, want <= %d", len(got), tt.maxBytes)
			}
			if !strings.HasSuffix(got, tt.wantEnd) && len(tt.input) > tt.maxBytes {
				t.Errorf("TruncateString() = %q, want suffix %q", got, tt.wantEnd)
			}
		})
	}
}

func TestTruncatePreview(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLen  int
		wantSuff string
	}{
		{
			name:     "short preview no truncation",
			input:    "hello world",
			wantLen:  11,
			wantSuff: "hello world",
		},
		{
			name:     "long preview truncated",
			input:    strings.Repeat("a", 2000),
			wantLen:  1003, // 1000 chars + "..."
			wantSuff: "...",
		},
		{
			name:     "exactly max chars",
			input:    strings.Repeat("x", MaxPreviewChars),
			wantLen:  MaxPreviewChars,
			wantSuff: strings.Repeat("x", 10), // last 10 chars
		},
		{
			name:     "utf8 preview",
			input:    strings.Repeat("世", 1500),
			wantLen:  1003, // 1000 runes + "..."
			wantSuff: "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TruncatePreview(tt.input)
			gotRunes := []rune(got)
			if len(gotRunes) > MaxPreviewChars+3 {
				t.Errorf("TruncatePreview() rune length = %d, want <= %d", len(gotRunes), MaxPreviewChars+3)
			}
			if !strings.HasSuffix(got, tt.wantSuff) {
				t.Errorf("TruncatePreview() = %q, want suffix %q", got, tt.wantSuff)
			}
		})
	}
}

func TestAttributeConstants(t *testing.T) {
	// Verify attribute constants are defined correctly
	constants := map[string]string{
		AttrSpanInputs:    "mlflow.spanInputs",
		AttrSpanOutputs:   "mlflow.spanOutputs",
		AttrSpanType:      "mlflow.spanType",
		AttrRequestID:     "mlflow.traceRequestId",
		AttrExperimentID:  "mlflow.experimentId",
		AttrTokenUsage:    "mlflow.chat.tokenUsage",
		AttrChatTools:     "mlflow.chat.tools",
		AttrMessageFormat: "mlflow.message.format",
		AttrLLMReasoning:  "mlflow.llm.reasoning",
		AttrFunctionName:  "mlflow.spanFunctionName",
		AttrLinkedPrompts: "mlflow.linkedPrompts",
	}

	for got, want := range constants {
		if got != want {
			t.Errorf("Constant mismatch: got %q, want %q", got, want)
		}
	}
}

func TestTruncationLimits(t *testing.T) {
	if MaxAttributeBytes != 10240 {
		t.Errorf("MaxAttributeBytes = %d, want 10240", MaxAttributeBytes)
	}
	if MaxPreviewChars != 1000 {
		t.Errorf("MaxPreviewChars = %d, want 1000", MaxPreviewChars)
	}
}
