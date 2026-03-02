package lingo

type LocalizationParams struct {
	SourceLocale *string                   `json:"source_locale,omitempty"`
	TargetLocale string                    `json:"target_locale"`
	Fast         *bool                     `json:"fast,omitempty"`
	Reference    map[string]map[string]any `json:"reference,omitempty"`
}
