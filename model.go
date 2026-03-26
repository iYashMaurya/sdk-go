package lingo

type LocalizationParams struct {
	SourceLocale *string                   `json:"source_locale,omitempty"`
	TargetLocale string                    `json:"target_locale"`
	Fast         *bool                     `json:"fast,omitempty"`
	Reference    map[string]map[string]any `json:"reference,omitempty"`
}

type RequestData struct {
	Param     parameter                 `json:"params"`
	Locale    locale                    `json:"locale"`
	Data      any                       `json:"data"`
	Reference map[string]map[string]any `json:"reference,omitempty"`
}

type parameter struct {
	WorkflowID string `json:"workflowId"`
	Fast       bool   `json:"fast"`
}

type locale struct {
	Source *string `json:"source,omitempty"`
	Target string  `json:"target"`
}
