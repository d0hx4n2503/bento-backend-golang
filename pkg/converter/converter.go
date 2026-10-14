package converter

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// AnyToBytesBuffer JSON-encodes value into a buffer without appending a newline.
func AnyToBytesBuffer(value any) (*bytes.Buffer, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal JSON: %w", err)
	}
	return bytes.NewBuffer(encoded), nil
}
