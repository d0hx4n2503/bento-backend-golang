package logger

import (
	"encoding/json"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

const consoleFieldKeyColor = "\x1b[96m"

type colorConsoleEncoder struct {
	zapcore.Encoder
}

func (e colorConsoleEncoder) Clone() zapcore.Encoder {
	return colorConsoleEncoder{Encoder: e.Encoder.Clone()}
}

func (e colorConsoleEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	encoded, err := e.Encoder.EncodeEntry(entry, fields)
	if err != nil {
		return nil, err
	}

	coloredContext := colorConsoleFieldKeys(encoded.Bytes())
	if coloredContext == nil {
		return encoded, nil
	}

	encoded.Reset()
	encoded.AppendBytes(coloredContext)
	return encoded, nil
}

func colorConsoleFieldKeys(line []byte) []byte {
	end := len(line)
	for end > 0 && (line[end-1] == '\n' || line[end-1] == '\r') {
		end--
	}

	for start := 0; start < end; start++ {
		if line[start] != '{' || (start > 0 && line[start-1] != '\t') {
			continue
		}
		if !json.Valid(line[start:end]) {
			continue
		}

		coloredJSON := colorJSONKeys(line[start:end])
		result := make([]byte, 0, len(line)+len(coloredJSON)-len(line[start:end]))
		result = append(result, line[:start]...)
		result = append(result, coloredJSON...)
		result = append(result, line[end:]...)
		return result
	}

	return nil
}

func colorJSONKeys(source []byte) []byte {
	result := make([]byte, 0, len(source)+32)
	for index := 0; index < len(source); {
		if source[index] != '"' {
			result = append(result, source[index])
			index++
			continue
		}

		stringEnd := jsonStringEnd(source, index)
		next := stringEnd + 1
		for next < len(source) && (source[next] == ' ' || source[next] == '\n' || source[next] == '\r' || source[next] == '\t') {
			next++
		}
		if next < len(source) && source[next] == ':' {
			result = append(result, consoleFieldKeyColor...)
			result = append(result, source[index:stringEnd+1]...)
			result = append(result, ansiReset...)
		} else {
			result = append(result, source[index:stringEnd+1]...)
		}
		index = stringEnd + 1
	}
	return result
}

func jsonStringEnd(source []byte, start int) int {
	for index := start + 1; index < len(source); index++ {
		switch source[index] {
		case '\\':
			index++
		case '"':
			return index
		}
	}
	return len(source) - 1
}
