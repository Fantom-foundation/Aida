// Package build provides build time information from the make process.
package build

import (
	"regexp"
	"testing"
)

func TestVersionInfoGivesOutput(t *testing.T) {
	expected := map[string]*regexp.Regexp{
		"App Version": regexp.MustCompile(`(?m)^\[\d+(;\d+)?mApp\sVersion:\[\d+(;\d+)?m\s+[\d.]+$`),
		"Commit Hash": regexp.MustCompile(`(?m)^\[\d+(;\d+)?mCommit\sHash:\[\d+(;\d+)?m\s+[a-f\d.]{40}$`),
		"Commit Time": regexp.MustCompile(`(?m)^\[\d+(;\d+)?mCommit\sTime:\[\d+(;\d+)?m\s+\w{3}, [0-9]{2} \w{3} [0-9]{4} [0-9]{2}:[0-9]{2}:[0-9]{2}(\s-?[0-9]{4})?$`),
		"Build Time":  regexp.MustCompile(`(?m)^\[\d+(;\d+)?mBuild\sTime:\[\d+(;\d+)?m\s+\w{3}, [0-9]{2} \w{3} [0-9]{4} [0-9]{2}:[0-9]{2}:[0-9]{2}(\s-?[0-9]{4})?$`),
		"Compiler":    regexp.MustCompile(`(?m)^\[\d+(;\d+)?mCompiler:\[\d+(;\d+)?m\s+\w+`),
	}

	v := VersionInfo()
	for n, r := range expected {
		if !r.MatchString(v) {
			t.Fatalf("VersionInfo() expected to contain %s information, returns:\n%s\n", n, v)
		}
	}
}
