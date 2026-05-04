package events

const (
	DeployRequested = "deploy_requested"
	BuildRequested  = "build_requested"
	SecurityScan    = "security_scan"
	PolicyCheck     = "policy_check"
	Deploy          = "deploy"
	Running         = "running"
	Failed          = "failed"
)

const ExchangeDeploys = "idp.deploys"
