package utils

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"go.uber.org/zap"

)

// Options for getVisualsTemplate
type options struct {
	AccessToken string `json:"accessToken"`
	ReportID    string `json:"reportId"`
}

const (
	visualsIndicator = "visuals_"
)

// GetVisuals returns a list of available visual for report
// TODO: Support multiple pages.
func GetVisuals(accessToken, reportID string) ([]string, error) {
	l := zap.L().With(zap.String("reportID", reportID))

	o := options{
		AccessToken: accessToken,
		ReportID:    reportID,
	}

	getVisualsTemplate := filepath.Join("resources", "getVisualsTemplate.html")
	reportHTMLPath, err := utils.GetEmbeddedReport(o.ReportID, getVisualsTemplate, "{{options}}", o)

	l.Info("HTML Path of the report", zap.String("path", reportHTMLPath))

	if err != nil {
		l.Error("couldn't build template", zap.Error(err))

		return nil, err
	}

	b, err := browser.GetBrowserInstance()
	if err != nil {
		l.Error("couldn't get browser instance", zap.Error(err))
		return nil, err
	}

	ctx, cancel := chromedp.NewContext(*b.GetContext())
	defer cancel()

	htmlReportAbsPath, err := utils.GetAbsolutePath(reportHTMLPath)

	defer func() {
		err = os.Remove(htmlReportAbsPath)
		if err != nil {
			l.Error("couldn't remove template", zap.Error(err), zap.String("reportPath", htmlReportAbsPath))
		}
	}()

	if err != nil {
		l.Error("couldn't build absolute path for template", zap.Error(err), zap.String("reportPath", reportHTMLPath))

		return nil, err
	}

	var consoleData string
	var consoleError error
	visualsChan := make(chan []string)
	defer close(visualsChan)

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *runtime.EventConsoleAPICalled: // watch for console.info()/console.error()
			stringValue := string(ev.Args[0].Value)
			if strings.Contains(stringValue, visualsIndicator) {
				consoleData = strings.Replace(stringValue, visualsIndicator, "", 1)
				consoleData = strings.Trim(consoleData, "\"")

				switch ev.Type {
				case runtime.APITypeInfo:
					if consoleData == "" {
						visualsChan <- []string{}
					} else {
						visualArray := strings.Split(consoleData, "\\n")
						visualsChan <- visualArray
					}

				case runtime.APITypeError:
					consoleError = errors.New("Get data of visual failed. " + consoleData)
					visualsChan <- []string{}
				}
			} else if strings.Contains(stringValue, reportErrorIndicator) {
				consoleError = domain.ErrReportNotLoaded
				visualsChan <- []string{}
			}
		}
	})

	openHTMLTask := chromedp.Tasks{chromedp.Navigate(filepath.Join("file:///", htmlReportAbsPath))}
	if err := chromedp.Run(ctx, openHTMLTask); err != nil {
		l.Error("couldn't open html file", zap.Error(err))
		return []string{}, err
	}

	l.Info("Console data: ", zap.String("Console data", consoleData))

	var visuals []string
	t := time.NewTimer(listenConsoleTimeout)
checkCondition:
	for {
		select {
		case visuals = <-visualsChan:
			break checkCondition

		case <-t.C:
			l.Error("condition checking timed out")

			return []string{}, errors.New("timeout exception")
		}
	}

	if consoleError != nil {
		l.Error("Console error", zap.String("Error", consoleError.Error()))

		return []string{}, consoleError
	}

	return visuals, nil
}
