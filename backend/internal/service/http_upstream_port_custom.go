package service

type HTTPUpstreamAccountIdleCloser interface {
	CloseIdleConnectionsForAccount(accountID int64)
}
