package loghelp

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

// ContextHook ...
type ContextHook struct{}

// Levels ...
func (hook ContextHook) Levels() []logrus.Level {
	return []logrus.Level{logrus.ErrorLevel}
}

// Fire ...
func (hook ContextHook) Fire(entry *logrus.Entry) error {
	for i := 5; i < 10; i++ {
		_, file, line, ok := runtime.Caller(i)
		if ok && strings.Contains(file, "prosper-pool") {
			trimmed := ShortenPoolFilePath(file, "", 0)

			entry.Data["source"] = fmt.Sprintf("%s:%v", trimmed, line)
			break
		}
	}

	return nil
}

// ShortenPoolFilePath takes a long path url to prosper-pool, and shortens it:
//	"/home/billy/go/src/github.com/FactomWyomingProject/prosper-pool/opr.go" -> "pegnet/opr.go"
//	This is nice for errors that print the file + line number
//
// 		!! Only use for error printing !!
//
func ShortenPoolFilePath(path, acc string, depth int) (trimmed string) {
	if depth > 5 || path == "." {
		// Recursive base case
		// If depth > 5 probably no prosper-pool dir exists
		return filepath.ToSlash(filepath.Join(path, acc))
	}
	dir, base := filepath.Split(path)
	if strings.ToLower(base) == "prosper-pool" { // Used to be named PegNet. Not everyone changed I bet
		return filepath.ToSlash(filepath.Join(base, acc))
	}

	return ShortenPoolFilePath(filepath.Clean(dir), filepath.Join(base, acc), depth+1)
}
