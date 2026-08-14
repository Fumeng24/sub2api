package repository

func init() {
	migrationChecksumCompatibilityRules["155_invoice_requests.sql"] =
		newMigrationChecksumCompatibilityRule(
			"d401e5393189fc8e57f5a74aa7aca6ca23537fb5a73673b9c16a81c9927dd52f",
			"f4fc2eba77594ad2ba77303dc965e0f5f27ae9144310e6bb0d69d6fddbbabd20",
		)
}
