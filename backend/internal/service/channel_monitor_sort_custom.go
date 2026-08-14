package service

func validateSortOrder(value int) error {
	if value < monitorMinSortOrder || value > monitorMaxSortOrder {
		return ErrChannelMonitorInvalidSortOrder
	}
	return nil
}
