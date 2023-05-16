package implementations



func removeUnused(allReportsBI domain.GroupedReports, usedReportIDs []string) domain.GroupedReports {
	newReportsBI := make(map[*domain.Group]*domain.ReportsContainer)

	for i, workspace := range allReportsBI {
		var reports []domain.IReport
		for _, report := range workspace.Value {
			if contains(usedReportIDs, report) {
				reports = append(reports, report)
			}
		}
		if len(reports) > 0 {
			workspace.Value = reports
			newReportsBI[i] = workspace
		}
	}

	return newReportsBI
}

func contains(dbReportIDs []string, reportPowerBI domain.IReport) bool {
	for _, rep := range dbReportIDs {
		if rep == reportPowerBI.GetID() {
			return true
		}
	}

	return false
}
