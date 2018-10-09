package selftest

import (
	"fmt"

	"github.com/dcos/dcos-diagnostics/dcos"
	log "github.com/sirupsen/logrus"
)

func findAgentsInHistoryServiceSelfTest(pastTime string) error {
	finder := &dcos.FindAgentsInHistoryService{PastTime: pastTime}
	nodes, err := finder.Find()
	if err != nil {
		return err
	}

	if len(nodes) == 0 {
		return fmt.Errorf("no nodes found in history service for past %s", pastTime)
	}

	return nil
}

func findAgentsInHistoryServicePastMinuteSelfTest() error {
	return findAgentsInHistoryServiceSelfTest("/minute/")
}

func findAgentsInHistoryServicePastHourSelfTest() error {
	return findAgentsInHistoryServiceSelfTest("/hour/")
}

func dummySelfTest() error {
	return nil
}

func getSelfTests() map[string]func() error {
	tests := make(map[string]func() error)
	tests["findAgentsInHistoryServicePastMinuteSelfTest"] = findAgentsInHistoryServicePastMinuteSelfTest
	tests["findAgentsInHistoryServicePastHourSelfTest"] = findAgentsInHistoryServicePastHourSelfTest
	tests["dummyTest"] = dummySelfTest
	return tests
}

// Result contains information about single test.
type Result struct {
	Success      bool
	ErrorMessage string
}

// Run performs selftest and returns map with results of different test with it's name as a key.
func Run() map[string]Result {
	result := make(map[string]Result)
	for selfTestName, fn := range getSelfTests() {
		err := fn()
		if err == nil {
			result[selfTestName] = Result{Success: true}
		} else {
			// check for NodesNotFoundError. Do not fail if this happens. It just means history service
			// was did not dump anything yet.
			if _, ok := err.(dcos.NodesNotFoundError); ok {
				log.WithError(err).Debug("Non critical error received")
				result[selfTestName] = Result{Success: true}
			} else {
				result[selfTestName] = Result{ErrorMessage: err.Error()}
			}
		}
	}
	return result
}
