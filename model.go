package lingo

// LocalizationParams contains parameters for a localization request.
// SourceLocale and Fast are optional pointer types — pass nil to use API defaults.
type LocalizationParams struct {
	SourceLocale *string                   `json:"source_locale,omitempty"`
	TargetLocale string                    `json:"target_locale"`
	Fast         *bool                     `json:"fast,omitempty"`
	Reference    map[string]map[string]any `json:"reference,omitempty"`
}

type requestData struct {
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
