package dynamic

import (
	"errors"
	"testing"
)

func TestClassifyTaskLabels(t *testing.T) {
	// @covers AC-AGENTS-OMNIROUTE-POLICY-001.3
	tests := []struct {
		name       string
		labelsJSON string
		want       TaskClassification
		wantErr    error
	}{
		{name: "missing defaults simple", labelsJSON: `[]`, want: TaskClassification{Class: TaskClassSimple, Source: "default"}},
		{name: "medium label", labelsJSON: `["bug","routing:medium"]`, want: TaskClassification{Class: TaskClassMedium, Source: "label"}},
		{name: "high risk label", labelsJSON: `["routing:high-risk"]`, want: TaskClassification{Class: TaskClassHighRisk, Source: "label"}},
		{name: "high complexity label", labelsJSON: `["routing:high-complexity"]`, want: TaskClassification{Class: TaskClassHighComplexity, Source: "label"}},
		{name: "conflicting labels", labelsJSON: `["routing:simple","routing:medium"]`, wantErr: ErrInvalidTaskClassification},
		{name: "malformed labels", labelsJSON: `{}`, wantErr: ErrInvalidTaskClassification},
		{name: "null is not a label array", labelsJSON: `null`, wantErr: ErrInvalidTaskClassification},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ClassifyTaskLabels(tt.labelsJSON)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("classification = %#v, want %#v", got, tt.want)
			}
		})
	}
}
