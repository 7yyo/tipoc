package mysql

import (
	"regexp"
	"strings"
)

func IsMysqlHeader(v string) bool {
	return strings.HasPrefix(v, mysqlCli)
}

func isRowInSetOutput(s string) bool {
	reg := regexp.MustCompile(`^\d+ rows? in set.*$`)
	return reg.MatchString(s)
}

func isQueryOKOutput(s string) bool {
	reg := regexp.MustCompile(`Query OK, \d+ rows? affected.*`)
	return reg.MatchString(s)
}

func isRecordsOutput(s string) bool {
	reg1 := regexp.MustCompile(`^Records: \d+  Duplicates: \d+  Warnings: \d+$`)
	reg2 := regexp.MustCompile(`^Records:\s+\d+\s+Deleted:\s+\d+\s+Skipped:\s+\d+\s+Warnings:\s+\d+$`)
	return reg1.MatchString(s) || reg2.MatchString(s)
}

func isRowsMatchedOutput(s string) bool {
	reg := regexp.MustCompile(`^Rows matched: \d+  Changed: \d+  Warnings: \d+$`)
	return reg.MatchString(s)
}

func isQueryOutput(s string) bool {
	return isRowInSetOutput(s) ||
		isQueryOKOutput(s) ||
		isErrorOutput(s)
}

func isDMLOutput(s string) bool {
	return isRecordsOutput(s) || isRowsMatchedOutput(s)
}

func isEmptySetOutput(s string) bool {
	reg := regexp.MustCompile("Empty set")
	return reg.MatchString(s)
}

func isErrorOutput(s string) bool {
	return strings.HasPrefix("ERROR", s)
}
