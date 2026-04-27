package llm

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var debugEnabled bool
var debugFile *os.File

// EnableDebug turns on debug logging to ~/.steered/debug.log
func EnableDebug() {
	debugEnabled = true
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".steered", "debug.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	debugFile = f
	debugLog("=== steered debug log started %s ===\n", time.Now().Format("2006-01-02 15:04:05"))
}

func debugLog(format string, args ...interface{}) {
	if !debugEnabled || debugFile == nil {
		return
	}
	fmt.Fprintf(debugFile, format, args...)
}

// DebugPrompt logs the full prompt sent to LLM
func DebugPrompt(prompt string) {
	debugLog("\n=== PROMPT ===\n%s\n=== END PROMPT ===\n", prompt)
}

// DebugResponse logs the full LLM response
func DebugResponse(response string) {
	debugLog("\n=== RESPONSE ===\n%s\n=== END RESPONSE ===\n", response)
}

// DebugIssues logs detected issues
func DebugIssues(issues []string) {
	debugLog("\n=== ISSUES ===\n")
	for _, i := range issues {
		debugLog("  %s\n", i)
	}
	debugLog("=== END ISSUES ===\n")
}
