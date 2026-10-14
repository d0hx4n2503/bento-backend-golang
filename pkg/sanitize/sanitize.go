package sanitize

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"

	"github.com/microcosm-cc/bluemonday"
)

var (
	ugcPolicy       = bluemonday.UGCPolicy()
	errTrailingJSON = errors.New("input contains more than one JSON value")
)

// SanitizeJSON removes unsafe HTML from every string value in a JSON document.
// Use this only for fields that are intentionally rendered as user-authored HTML;
// it changes string data and is not a substitute for context-aware output escaping.
func SanitizeJSON(input []byte) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}

	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err != nil {
			return nil, err
		}
		return nil, errTrailingJSON
	}

	return json.MarshalIndent(sanitizeValue(value), "", "    ")
}

func sanitizeValue(value any) any {
	switch typed := value.(type) {
	case string:
		return ugcPolicy.Sanitize(typed)
	case []any:
		for i := range typed {
			typed[i] = sanitizeValue(typed[i])
		}
		return typed
	case map[string]any:
		for key, item := range typed {
			typed[key] = sanitizeValue(item)
		}
		return typed
	default:
		return value
	}
}
