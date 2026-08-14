package apicompat

import "encoding/json"

func (a *WebSearchAction) UnmarshalJSON(data []byte) error {
	if a == nil {
		return nil
	}

	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	switch value := raw.(type) {
	case nil:
		*a = WebSearchAction{}
	case string:
		*a = WebSearchAction{Type: value}
	case map[string]any:
		type alias WebSearchAction
		var parsed alias
		if err := json.Unmarshal(data, &parsed); err != nil {
			return err
		}
		*a = WebSearchAction(parsed)
	default:
		*a = WebSearchAction{}
	}
	return nil
}
