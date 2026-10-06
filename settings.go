package commentratio

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

// ErrInvalidSettings is returned when settings cannot be decoded or contain invalid values.
var ErrInvalidSettings = errors.New("invalid settings")

// Rule holds the limits for one check target.
type Rule struct {
	Enabled   bool    `json:"enabled"`
	FreeLines int     `json:"free-lines"`
	MaxRatio  float64 `json:"max-ratio"`
	MaxLines  int     `json:"max-lines"`
}

// DocRule is a Rule for doc comments, which can also require a doc comment.
// RequireFrom of 0 disables the missing doc check.
type DocRule struct {
	Rule

	RequireFrom int `json:"require-from"`
}

// Settings holds the rules for all check targets.
type Settings struct {
	FuncDoc  DocRule `json:"func-doc"`
	DeclDoc  DocRule `json:"decl-doc"`
	FuncBody Rule    `json:"func-body"`
	File     Rule    `json:"file"`
}

const (
	keyFuncDoc  = "func-doc"
	keyDeclDoc  = "decl-doc"
	keyFuncBody = "func-body"
	keyFile     = "file"

	keyFreeLines   = "free-lines"
	keyMaxRatio    = "max-ratio"
	keyMaxLines    = "max-lines"
	keyRequireFrom = "require-from"
)

const (
	defaultDocFreeLines   = 3
	defaultDocMaxRatio    = 0.2
	defaultDocMaxLines    = 15
	defaultDocRequireFrom = 10

	defaultFuncBodyFreeLines = 2
	defaultFuncBodyMaxRatio  = 0.2
	defaultFuncBodyMaxLines  = 10

	defaultFileFreeLines = 5
	defaultFileMaxRatio  = 0.3
	defaultFileMaxLines  = 200
)

// DefaultSettings returns the settings used when nothing is configured.
func DefaultSettings() Settings {
	docRule := DocRule{
		Rule: Rule{
			Enabled:   true,
			FreeLines: defaultDocFreeLines,
			MaxRatio:  defaultDocMaxRatio,
			MaxLines:  defaultDocMaxLines,
		},
		RequireFrom: defaultDocRequireFrom,
	}

	return Settings{
		FuncDoc: docRule,
		DeclDoc: docRule,
		FuncBody: Rule{
			Enabled:   true,
			FreeLines: defaultFuncBodyFreeLines,
			MaxRatio:  defaultFuncBodyMaxRatio,
			MaxLines:  defaultFuncBodyMaxLines,
		},
		File: Rule{
			Enabled:   true,
			FreeLines: defaultFileFreeLines,
			MaxRatio:  defaultFileMaxRatio,
			MaxLines:  defaultFileMaxLines,
		},
	}
}

// Validate returns an error wrapping ErrInvalidSettings if any rule, enabled or not, has an invalid value.
func (s Settings) Validate() error {
	if err := s.FuncDoc.validate(keyFuncDoc); err != nil {
		return err
	}

	if err := s.DeclDoc.validate(keyDeclDoc); err != nil {
		return err
	}

	if err := s.FuncBody.validate(keyFuncBody); err != nil {
		return err
	}

	return s.File.validate(keyFile)
}

func (r DocRule) validate(name string) error {
	if r.RequireFrom < 0 {
		return fmt.Errorf("%w: %s.%s must be >= 0, got %d", ErrInvalidSettings, name, keyRequireFrom, r.RequireFrom)
	}

	return r.Rule.validate(name)
}

func (r Rule) validate(name string) error {
	if r.FreeLines < 0 {
		return fmt.Errorf("%w: %s.%s must be >= 0, got %d", ErrInvalidSettings, name, keyFreeLines, r.FreeLines)
	}

	if r.MaxLines < 0 {
		return fmt.Errorf("%w: %s.%s must be >= 0, got %d", ErrInvalidSettings, name, keyMaxLines, r.MaxLines)
	}

	if math.IsNaN(r.MaxRatio) || math.IsInf(r.MaxRatio, 0) || r.MaxRatio < 0 {
		return fmt.Errorf("%w: %s.%s must be a finite number >= 0, got %v", ErrInvalidSettings, name, keyMaxRatio, r.MaxRatio)
	}

	if r.FreeLines > r.MaxLines {
		return fmt.Errorf("%w: %s.%s (%d) must be <= %s.%s (%d)",
			ErrInvalidSettings, name, keyFreeLines, r.FreeLines, name, keyMaxLines, r.MaxLines)
	}

	return nil
}

// DecodeSettings decodes golangci-lint plugin settings on top of DefaultSettings and validates the result.
// Omitted keys and null values keep their default values. Keys are matched as in encoding/json, which ignores case,
// and keys that match no field are rejected.
func DecodeSettings(raw any) (Settings, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return Settings{}, fmt.Errorf("%w: encode: %w", ErrInvalidSettings, err)
	}

	s := DefaultSettings()
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	if err := dec.Decode(&s); err != nil {
		return Settings{}, fmt.Errorf("%w: decode: %w", ErrInvalidSettings, err)
	}

	if err := s.Validate(); err != nil {
		return Settings{}, err
	}

	return s, nil
}
