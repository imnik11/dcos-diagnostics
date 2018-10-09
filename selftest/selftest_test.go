package selftest

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRun_withFailingHistoryService(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Errors on windows has different body")
	}
	result := Run()
	expected := map[string]Result{
		"dummyTest": {Success: true},
		"findAgentsInHistoryServicePastHourSelfTest":   {ErrorMessage: "open /var/lib/dcos/dcos-history/hour/: no such file or directory"},
		"findAgentsInHistoryServicePastMinuteSelfTest": {ErrorMessage: "open /var/lib/dcos/dcos-history/minute/: no such file or directory"},
	}
	assert.Equal(t, expected, result)
}
