package service

import "encoding/json"

func defaultTicketSystemSettingsJSON() string {
	payload, err := json.Marshal(DefaultTicketSystemSettings())
	if err != nil {
		return `{"templates":[],"support_permissions":{},"sla":{"enabled":true,"first_response_minutes":1440,"reminder_before_minutes":60,"auto_escalate_after_minutes":0,"reminder_notifications":true,"auto_escalate_notifications":true,"auto_close_resolved_days":0,"worker_interval_seconds":300}}`
	}
	return string(payload)
}
