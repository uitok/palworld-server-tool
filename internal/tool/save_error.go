package tool

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

type SaveSourceKind string

const (
	SaveSourceLocal  SaveSourceKind = "local"
	SaveSourceHTTP   SaveSourceKind = "http"
	SaveSourceDocker SaveSourceKind = "docker"
	SaveSourceK8s    SaveSourceKind = "k8s"
)

type SaveOperationError struct {
	Code       string
	Operation  string
	Stage      string
	SourceKind SaveSourceKind
	SourcePath string
	Err        error
	message    string
}

func (e *SaveOperationError) Error() string {
	if e == nil {
		return ""
	}
	if e.message != "" {
		return e.message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Code
}

func (e *SaveOperationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func SaveOperationErrorCode(err error) string {
	var opErr *SaveOperationError
	if errors.As(err, &opErr) {
		return opErr.Code
	}
	return ""
}

func SaveOperationErrorDetails(err error) map[string]any {
	var opErr *SaveOperationError
	if !errors.As(err, &opErr) || opErr == nil {
		return nil
	}
	return map[string]any{
		"operation":   opErr.Operation,
		"stage":       opErr.Stage,
		"source_kind": opErr.SourceKind,
		"source_path": opErr.SourcePath,
		"code":        opErr.Code,
	}
}

func detectSaveSourceKind(path string) SaveSourceKind {
	trimmed := strings.TrimSpace(path)
	switch {
	case strings.HasPrefix(trimmed, "http://"), strings.HasPrefix(trimmed, "https://"):
		return SaveSourceHTTP
	case strings.HasPrefix(trimmed, "docker://"):
		return SaveSourceDocker
	case strings.HasPrefix(trimmed, "k8s://"):
		return SaveSourceK8s
	default:
		return SaveSourceLocal
	}
}

func wrapSaveSourceError(path, stage string, err error) error {
	if err == nil {
		return nil
	}
	var opErr *SaveOperationError
	if errors.As(err, &opErr) {
		return err
	}
	kind := detectSaveSourceKind(path)
	code, message := classifySaveSourceError(kind, stage, err)
	return &SaveOperationError{
		Code:       code,
		Operation:  "source",
		Stage:      stage,
		SourceKind: kind,
		SourcePath: path,
		Err:        err,
		message:    message,
	}
}

func wrapSaveDecodeError(path, stage string, err error) error {
	if err == nil {
		return nil
	}
	var opErr *SaveOperationError
	if errors.As(err, &opErr) {
		return err
	}
	kind := detectSaveSourceKind(path)
	code, message := classifySaveDecodeError(kind, stage, err)
	return &SaveOperationError{
		Code:       code,
		Operation:  "decode",
		Stage:      stage,
		SourceKind: kind,
		SourcePath: path,
		Err:        err,
		message:    message,
	}
}

func classifySaveSourceError(kind SaveSourceKind, stage string, err error) (string, string) {
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	label := string(kind)
	if label == "" {
		label = "save"
	}
	switch {
	case stage == "parse":
		return "save_source_invalid", fmt.Sprintf("invalid %s save source address: %v", label, err)
	case errors.Is(err, os.ErrNotExist):
		return "save_source_not_found", fmt.Sprintf("%s save source not found: %v", label, err)
	case strings.Contains(message, "specified file is not level.sav"):
		return "save_source_invalid", fmt.Sprintf("invalid %s save source: %v", label, err)
	case strings.Contains(message, "status 404"):
		return "save_source_not_found", fmt.Sprintf("%s save source not found: %v", label, err)
	case strings.Contains(message, "directory containing level.sav not found"):
		return "save_source_not_found", fmt.Sprintf("%s save source not found: %v", label, err)
	case kind == SaveSourceHTTP && strings.Contains(message, "level.sav not found"):
		return "save_source_invalid", fmt.Sprintf("invalid http save source content: %v", err)
	case isSaveSourceUnreachable(err):
		return "save_source_unreachable", fmt.Sprintf("%s save source unreachable: %v", label, err)
	default:
		return "save_source_copy_failed", fmt.Sprintf("failed to read %s save source: %v", label, err)
	}
}

func classifySaveDecodeError(kind SaveSourceKind, stage string, err error) (string, string) {
	label := string(kind)
	if label == "" {
		label = "save"
	}
	switch stage {
	case "cli":
		if errors.Is(err, os.ErrNotExist) {
			return "save_decode_cli_missing", fmt.Sprintf("sav_cli not found: %v", err)
		}
		return "save_decode_prepare_failed", fmt.Sprintf("failed to prepare sav decode tool: %v", err)
	case "token":
		return "save_decode_prepare_failed", fmt.Sprintf("failed to prepare save decode request: %v", err)
	default:
		return "save_decode_failed", fmt.Sprintf("failed to decode save data from %s source: %v", label, err)
	}
}

func isSaveSourceUnreachable(err error) bool {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "connection refused") ||
		strings.Contains(message, "no such host") ||
		strings.Contains(message, "timeout") ||
		strings.Contains(message, "error getting in-cluster config") ||
		strings.Contains(message, "error getting clientset")
}
