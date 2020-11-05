package controllers

import "github.com/go-logr/logr"
import "testing"

type FakeScalingDecisionService struct {
	FakeDecision *ScalingDecision
}

func (s FakeScalingDecisionService) scalingDecision(name string, min int32, current int32) (*ScalingDecision, error) {
	//println(fmt.Printf("-------------object ref: %v" , s.FakeDecision))
	return s.FakeDecision, nil
}

/**
This is a dummy logger implementation, allows to turn on info logging for debugging purposes on tests.
*/
type TestLogger struct {
	T       *testing.T
	LogInfo bool
}

func (log TestLogger) Info(msg string, args ...interface{}) {
	if log.LogInfo {
		log.T.Logf("%s -- %v", msg, args)
	}
}

func (_ TestLogger) Enabled() bool {
	return false
}

func (log TestLogger) Error(err error, msg string, args ...interface{}) {
	log.T.Logf("%s: %v -- %v", msg, err, args)
}

func (log TestLogger) V(v int) logr.InfoLogger {
	return log
}

func (log TestLogger) WithName(_ string) logr.Logger {
	return log
}

func (log TestLogger) WithValues(_ ...interface{}) logr.Logger {
	return log
}
