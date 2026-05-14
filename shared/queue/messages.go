package queue

import (
	"time"

	"github.com/ixcsoft/idp/shared/spec"
)

type WorkMessage struct {
	DeployID string      `json:"deploy_id"`
	Payload  spec.Intent `json:"payload"`
}

type StatusEvent struct {
	DeployID  string    `json:"deploy_id"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	URL       string    `json:"url,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

const (
	StatusQueued         = "queued"
	StatusSecurityPassed = "security_passed"
	StatusPolicyPassed   = "policy_passed"
	StatusRunning        = "running"
	StatusFailed         = "failed"
)
