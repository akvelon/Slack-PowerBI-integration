package browser

import (
	"context"
	"io/ioutil"
	"math"
	"net/url"
	"path/filepath"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"go.uber.org/zap"

)

const (
	generatedScreenshotsPath string = "./generatedReportsScreenshots"
)

// ScreenshotOptions represents options for GetScreenshot func
type ScreenshotOptions struct {
	BrowserContext     context.Context
	HTLMlFilePath      string
	RenderedIndicator  string // screenshot is captured once the the element with id = <RenderedIndicator> is visible
	ScreenshotFileName string
}

// GetScreenshot take a screenshot of html file
func GetScreenshot(o *ScreenshotOptions) (string, error) {
	l := zap.L()

	err := utils.CheckAndCreateDir(generatedScreenshotsPath)
	if err != nil {
		l.Error("couldn't create screenshots directory", zap.Error(err))

		return "", err
	}

	screenshotDirAbsPath, err := utils.GetAbsolutePath(generatedScreenshotsPath)
	if err != nil {
		l.Error("couldn't build screenshot path", zap.Error(err))

		return "", err
	}

	htmlAbsPath, err := utils.GetAbsolutePath(o.HTLMlFilePath)
	if err != nil {
		l.Error("couldn't build html path", zap.Error(err))

		return "", err
	}

	var buf []byte

	var origCtx context.Context
	if o.BrowserContext != nil {
		origCtx = o.BrowserContext
	} else {
		browser, err := GetBrowserInstance()
		if err != nil {
			l.Error("couldn't get browser instance", zap.Error(err))
			return "", err
		}

		origCtx = *browser.GetContext()
	}

	ctx, cancel := chromedp.NewContext(origCtx)
	defer cancel()

	err = chromedp.Run(ctx, fullScreenshot(filepath.Join("file:///", htmlAbsPath), o.RenderedIndicator, 100, &buf))
	if err != nil {
		l.Error("browser failed", zap.Error(err), zap.String("reportPath", htmlAbsPath))

		return "", err
	}

	screenshotName := utils.GetUniqueFileName(o.ScreenshotFileName, "png")
	screenshotFullName := filepath.Join(screenshotDirAbsPath, screenshotName)
	err = ioutil.WriteFile(screenshotFullName, buf, 0644)
	if err != nil {
		l.Error("couldn't write screenshot", zap.Error(err), zap.String("screenshotPath", screenshotFullName))

		return "", err
	}

	return screenshotFullName, nil
}

// fullScreenshot takes a screenshot of the entire browser viewport
func fullScreenshot(urlstr string, renderedMark string, quality int64, res *[]byte) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			var currentURL string
			_ = chromedp.Location(&currentURL).Do(ctx)
			c, err := url.Parse(currentURL)
			if err != nil {
				return err
			}

			u, _ := url.Parse(urlstr)
			if err != nil {
				return err
			}

			if *c != *u {
				_ = chromedp.Navigate(urlstr).Do(ctx)
			}

			return nil
		}),
		chromedp.WaitVisible(renderedMark, chromedp.ByID),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// get layout metrics
			_, _, contentSize, _, _, _, err := page.GetLayoutMetrics().Do(ctx)
			if err != nil {
				return err
			}

			width, height := int64(math.Ceil(contentSize.Width)), int64(math.Ceil(contentSize.Height))

			// force viewport emulation
			err = emulation.SetDeviceMetricsOverride(width, height, 1, false).
				WithScreenOrientation(&emulation.ScreenOrientation{
					Type:  emulation.OrientationTypePortraitPrimary,
					Angle: 0,
				}).
				Do(ctx)
			if err != nil {
				return err
			}

			// capture screenshot
			*res, err = page.CaptureScreenshot().
				WithQuality(quality).
				WithClip(&page.Viewport{
					X:      contentSize.X,
					Y:      contentSize.Y,
					Width:  contentSize.Width,
					Height: contentSize.Height,
					Scale:  1,
				}).Do(ctx)
			if err != nil {
				return err
			}

			return nil
		}),
	}
}
