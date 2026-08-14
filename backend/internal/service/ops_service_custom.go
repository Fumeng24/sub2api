package service

import "github.com/Wei-Shaw/sub2api/internal/util/logredact"

type opsServiceCustom struct {
	runtimeAlertService *OpsRuntimeAlertService
}

func (s *OpsService) initRuntimeAlertServiceCustom(services ...*OpsRuntimeAlertService) {
	if len(services) > 0 {
		s.runtimeAlertService = services[0]
	}
	if s.runtimeAlertService != nil {
		s.runtimeAlertService.opsService = s
	}
}

func (s *OpsService) SetRuntimeAlertService(runtimeAlertService *OpsRuntimeAlertService) {
	if s == nil {
		return
	}
	s.runtimeAlertService = runtimeAlertService
	if runtimeAlertService != nil {
		runtimeAlertService.opsService = s
	}
}

func isSensitiveKeyCustom(key string) bool {
	switch key {
	case "api-key", "accesstoken", "refreshtoken", "idtoken", "sessiontoken", "clientsecret":
		return true
	default:
		return false
	}
}

func redactOpsNonJSONErrorBodyCustom(raw string) string {
	return logredact.RedactText(raw)
}
