package dynamic

import (
	"encoding/json"
	"errors"
	"fmt"
)

type TaskClass string

const (
	TaskClassSimple         TaskClass = "simple"
	TaskClassMedium         TaskClass = "medium"
	TaskClassHighRisk       TaskClass = "high_risk"
	TaskClassHighComplexity TaskClass = "high_complexity"
)

var ErrInvalidTaskClassification = errors.New("invalid task routing classification")

type TaskClassification struct {
	Class  TaskClass
	Source string
}

func ClassifyTaskLabels(raw string) (TaskClassification, error) {
	var labels []string
	if err := json.Unmarshal([]byte(raw), &labels); err != nil {
		return TaskClassification{}, fmt.Errorf("%w: labels must be a JSON string array", ErrInvalidTaskClassification)
	}
	if labels == nil {
		return TaskClassification{}, fmt.Errorf("%w: labels must be a JSON string array", ErrInvalidTaskClassification)
	}
	classes := map[string]TaskClass{
		"routing:simple":          TaskClassSimple,
		"routing:medium":          TaskClassMedium,
		"routing:high-risk":       TaskClassHighRisk,
		"routing:high-complexity": TaskClassHighComplexity,
	}
	var selected TaskClass
	for _, label := range labels {
		class, ok := classes[label]
		if !ok {
			continue
		}
		if selected != "" && selected != class {
			return TaskClassification{}, fmt.Errorf("%w: conflicting routing labels", ErrInvalidTaskClassification)
		}
		selected = class
	}
	if selected == "" {
		return TaskClassification{Class: TaskClassSimple, Source: "default"}, nil
	}
	return TaskClassification{Class: selected, Source: "label"}, nil
}
