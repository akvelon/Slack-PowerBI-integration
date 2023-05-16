package utils

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"go.uber.org/zap"
r"
)

// Options for alertTemplate
type Options struct {
	AccessToken string  `json:"accessToken"`
	ReportID    string  `json:"reportId"`
	VisualName  string  `json:"visualName"`
	Threshold   float64 `json:"threshold"`
	Condition   string  `json:"condition"`
}

const (
	alertAnalysisIndicator = "alertAnalysis_"
	reportErrorIndicator   = "reportError_"
)

const listenConsoleTimeout time.Duration = 30 * time.Second

// CheckAlertAndTakeScreenShot take a screenshot if threshold is exceeded
// TODO: Support multiple pages.
func CheckAlertAndTakeScreenShot(accessToken, reportID, visualName string, threshold float64, condition string) (string, error) {
	l := zap.L().With(zap.String("reportID", reportID))

	l.Info("checking alert")

	options := Options{
		AccessToken: accessToken,
		ReportID:    reportID,
		VisualName:  visualName,
		Threshold:   threshold,
		Condition:   condition,
	}
	checkAlertTemplate := filepath.Join("resources", "checkAlertTemplate.html")
	reportHTMLPath, err := utils.GetEmbeddedReport(options.ReportID, checkAlertTemplate, "{{options}}", options)

	if err != nil {
		l.Error("couldn't build template", zap.Error(err))

		return "", err
	}

	b, err := browser.GetBrowserInstance()
	if err != nil {
		l.Error("couldn't get browser instance", zap.Error(err))
		return "", err
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

		return "", err
	}

	isThresholdExceeded, err := doesVisualDataExceedThreshold(ctx, htmlReportAbsPath, &options)
	if err != nil {
		l.Error("couldn't proceed does visual data exceed threshold", zap.Error(err), zap.String("reportPath", reportHTMLPath))

		return "", err
	}

	if !isThresholdExceeded {
		l.Info("no threshold(-s) exceeded", zap.String("reportPath", reportHTMLPath))

		return "", nil
	}

	l.Info("threshold exceeded", zap.String("reportID", reportID), zap.String("visualName", visualName))

	o := &browser.ScreenshotOptions{
		BrowserContext:     ctx,
		HTLMlFilePath:      htmlReportAbsPath,
		RenderedIndicator:  "#done",
		ScreenshotFileName: reportID,
	}

	return browser.GetScreenshot(o)
}

func doesVisualDataExceedThreshold(ctx context.Context, embeddedReportPath string, options *Options) (bool, error) {
	l := zap.L().With(zap.String("reportID", options.ReportID))
	var consoleData string
	var consoleError error
	doScreenShot := make(chan bool)
	defer close(doScreenShot)

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *runtime.EventConsoleAPICalled: // watch for console.info()/console.error()
			stringValue := string(ev.Args[0].Value)
			if strings.Contains(stringValue, alertAnalysisIndicator) {
				consoleData = strings.Replace(stringValue, alertAnalysisIndicator, "", 1)
				consoleData = strings.Trim(consoleData, "\"")
				switch ev.Type {
				case runtime.APITypeInfo:
					var isThresholdExceeded bool
					isThresholdExceeded, consoleError = isThresholdExceededFn(consoleData, options.Condition, options.Threshold)
					doScreenShot <- isThresholdExceeded

				case runtime.APITypeError:
					consoleError = errors.New("Get data of visual failed. " + consoleData)
					doScreenShot <- false
				}
			} else if strings.Contains(stringValue, reportErrorIndicator) {
				consoleError = domain.ErrReportNotLoaded
				doScreenShot <- false
			}
		}
	})

	openHTMLTask := chromedp.Tasks{chromedp.Navigate(filepath.Join("file:///", embeddedReportPath))}
	if err := chromedp.Run(ctx, openHTMLTask); err != nil {
		l.Error("couldn't open html file", zap.Error(err))
		return false, err
	}

	t := time.NewTimer(listenConsoleTimeout)
checkCondition:
	for {
		select {
		case shouldDoScreenShot := <-doScreenShot:
			if !shouldDoScreenShot {
				if consoleError != nil {
					l.Error("console error", zap.Error(consoleError))
					return false, consoleError
				}

				return false, nil
			}

			break checkCondition

		case <-t.C:
			l.Error("condition checking timed out")

			return false, errors.New("timeout exception")
		}
	}

	return true, nil
}

func isThresholdExceededFn(value string, condition string, threshold float64) (bool, error) {
	dataOfVisualInt, err := strconv.ParseFloat(value, 64)
	if err != nil {
		getDataOfVisualErr := errors.New("visual data is not a number")

		return false, getDataOfVisualErr
	}

	switch condition {
	case "below":
		return dataOfVisualInt < threshold, nil

	case "above":
		return dataOfVisualInt > threshold, nil

	case "equal":
		return dataOfVisualInt == threshold, nil
	}

	return false, errors.New("unknown condition")
}
