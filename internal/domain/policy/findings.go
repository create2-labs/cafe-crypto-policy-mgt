package policy

func blockingFinding(code, message string) AssessmentFinding {
	return AssessmentFinding{Code: code, Message: message, Severity: AssessmentFindingSeverityBlocking}
}

func fieldFinding(code, message, field string) AssessmentFinding {
	finding := blockingFinding(code, message)
	finding.Field = field
	return finding
}

