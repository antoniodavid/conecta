package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Envelope is the single machine-readable document emitted on stdout.
// Success: {ok:true,data:...}. Failure: {ok:false,error:{code,message}}.
type Envelope struct {
	Ok    bool            `json:"ok"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error *ErrBody        `json:"error,omitempty"`
}

// ErrBody carries a stable machine-readable code plus a human message.
type ErrBody struct {
	Code string `json:"code"` // invalid_input|config|unavailable|op_failed|authz
	Msg  string `json:"message"`
}

// exitForCode maps contract error codes to deterministic process exits:
// 0 ok, 2 invalid input/config, 3 unavailable/operation failure, 4 authorization failure.
func exitForCode(code string) int {
	switch code {
	case "invalid_input", "config":
		return 2
	case "unavailable", "op_failed":
		return 3
	case "authz":
		return 4
	default:
		return 3
	}
}

func mustMarshal(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		// Must never emit multi-line or invalid JSON; fall back to a minimal failure doc.
		return `{"ok":false,"error":{"code":"op_failed","message":"encoding failure"}}`
	}
	return string(b)
}

// buildResult returns a compact single-line success document (no trailing newline).
func buildResult(data any) string {
	var raw json.RawMessage
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			out, _ := buildError("op_failed", "encoding failure")
			return out
		}
		raw = b
	} else {
		raw = json.RawMessage(`{}`)
	}
	return mustMarshal(Envelope{Ok: true, Data: raw})
}

// buildError returns a compact single-line failure document plus its exit code.
func buildError(code, msg string) (string, int) {
	if code == "" {
		code = "op_failed"
	}
	if msg == "" {
		msg = code
	}
	return mustMarshal(Envelope{Ok: false, Error: &ErrBody{Code: code, Msg: msg}}), exitForCode(code)
}

// emitResult writes one JSON line to stdout; diagnostics stay on stderr.
func emitResult(data any) {
	fmt.Fprintln(os.Stdout, buildResult(data))
}

// emitError writes one JSON failure line to stdout and exits with the mapped code.
func emitError(code, msg string) {
	out, exit := buildError(code, msg)
	fmt.Fprintln(os.Stdout, out)
	os.Exit(exit)
}
