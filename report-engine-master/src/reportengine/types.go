package reportengine

type pageScreenshot struct {
	pageID   string
	pageName string
	rawData  []byte
}

type resource string

const (
	resourceReportTemplate2 resource = "reportTemplate2.html"
)
