package output

import (
	"encoding/json"
	"fmt"
	"os"
)

var JSONMode bool

func SetJSONMode(enabled bool) {
	JSONMode = enabled
}

func IsJSONMode() bool {
	return JSONMode
}

func PrintJSON(data interface{}) error {
	envelope := Envelope{
		Version: APIVersion,
		Data:    data,
	}
	return writeJSON(envelope)
}

func PrintError(code, message string) error {
	envelope := ErrorEnvelope{
		Version: APIVersion,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	}
	return writeJSON(envelope)
}

func writeJSON(v interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func FormatProjectList(projects []ProjectJSON) error {
	if !JSONMode {
		return nil
	}
	return PrintJSON(&PSData{})
}

func FormatPS(containers []ContainerJSON) error {
	if !JSONMode {
		return nil
	}
	data := &PSData{
		Containers: containers,
	}
	return PrintJSON(data)
}

func FormatError(err error) error {
	if !JSONMode {
		return nil
	}
	return PrintError("UNKNOWN", err.Error())
}

func FormatErrorCode(code, message string) error {
	if !JSONMode {
		return nil
	}
	return PrintError(code, message)
}

func PrintSuccess(message string) error {
	if !JSONMode {
		fmt.Println(message)
		return nil
	}
	return nil
}
