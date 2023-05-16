package reportengine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"go.uber.org/zap"


)

// CDPEngine is a ReportEngine powered by a Chrome instance controlled via Chrome DevTools Protocol.
type CDPEngine struct {
	config            *config.BrowserConfig
	logger            *zap.Logger
	allocatorOptions  []chromedp.ExecAllocatorOption
	allocatorCtx      context.Context
	browserCtxOptions []chromedp.ContextOption
	browserCtx        context.Context
}

// NewCDPReportEngine creates a Chrome-based ReportEngine.
func NewCDPReportEngine(c *config.BrowserConfig, l *zap.Logger) *CDPEngine {
	allocatorOptions := chromedp.DefaultExecAllocatorOptions[:]
	if !c.Headless {
		allocatorOptions = append(allocatorOptions, noHeadless)
	}

	browserCtxOptions := []chromedp.ContextOption(nil)
	if c.RedirectLog {
		l := newZapCDPAdapter(l.Named("cdp"))
		browserOptions := []chromedp.BrowserOption{
			chromedp.WithBrowserErrorf(l.Log),
			chromedp.WithBrowserLogf(l.Log),
			chromedp.WithBrowserDebugf(l.Log),
		}
		browserCtxOptions = append(browserCtxOptions, chromedp.WithBrowserOption(browserOptions...))
	}

	return &CDPEngine{
		config:            c,
		logger:            l,
		allocatorOptions:  allocatorOptions,
		browserCtxOptions: browserCtxOptions,
	}
}

// Start preconfigures a CDPEngine.
func (e *CDPEngine) Start(ctx context.Context) error {
	e.allocatorCtx, _ = chromedp.NewExecAllocator(ctx, e.allocatorOptions...)

	return e.startBrowser()
}

// Stop releases resources held by CDPEngine.
func (e *CDPEngine) Stop() error {
	return chromedp.Cancel(e.browserCtx)
}

// NewContext creates a context.Context suitable to pass to other methods.
func (e *CDPEngine) NewContext() (context.Context, context.CancelFunc, error) {
	// NOTE: To ensure stable behavior, we attempt to revive browser process if it had died in the meantime.
	err := e.browserCtx.Err()
	if err == context.Canceled {
		err2 := e.startBrowser()
		if err2 != nil {
			return nil, nil, err2
		}
	} else if err != nil {
		return nil, nil, err
	}

	timeoutCtx, cancelTimeout := context.WithTimeout(e.browserCtx, e.config.TabTimeout)
	tabCtx, _ := chromedp.NewContext(timeoutCtx)

	return tabCtx, cancelTimeout, nil
}

// RenderReport renders a report into set of images for each page chosen.
func (e *CDPEngine) RenderReport(ctx context.Context, o *utils.ShareOptions) (*RenderedReport, error) {
	template, err := e.newRenderReportTemplate(resourceReportTemplate2)
	if err != nil {
		return nil, err
	}

	screenshots := []*pageScreenshot(nil)
	takeScreenshots := e.newScreenshotPagesTask(ctx, &screenshots, template, o)
	err = chromedp.Run(ctx, takeScreenshots)
	if err != nil {
		return nil, err
	}

	timestamp := timestamp(ctx)
	pages := []*RenderedPage(nil)
	for _, pageScreenshot := range screenshots {
		filename := ""
		if o.Filter != nil {
			filename = fmt.Sprintf("%v (%v): %v %v.png", o.ReportName, o.Filter.String(), pageScreenshot.pageName, timestamp)
		} else {
			filename = fmt.Sprintf("%v: %v %v.png", o.ReportName, pageScreenshot.pageName, timestamp)
		}

		renderedPage := RenderedPage{
			ID:        pageScreenshot.pageID,
			Name:      pageScreenshot.pageName,
			Filename:  filename,
			ImageData: pageScreenshot.rawData,
		}
		pages = append(pages, &renderedPage)
	}

	return &RenderedReport{
		ID:    o.ReportID,
		Name:  o.ReportName,
		Pages: pages,
	}, nil
}

func (e *CDPEngine) startBrowser() error {
	e.browserCtx, _ = chromedp.NewContext(e.allocatorCtx, e.browserCtxOptions...)

	return chromedp.Run(e.browserCtx)
}

func (e *CDPEngine) newRenderReportTemplate(r resource) (url.URL, error) {
	relativeResourcePath := filepath.Join(e.config.ResourcesDirectory, string(r))

	_, err := os.Stat(relativeResourcePath)
	if err != nil {
		return url.URL{}, err
	}

	absoluteFilePath, err := filepath.Abs(relativeResourcePath)
	if err != nil {
		return url.URL{}, err
	}

	return url.URL{
		Scheme: "file",
		Path:   "/" + absoluteFilePath,
	}, nil
}

// TODO: Collect resource usage metrics, see `https://chromedevtools.github.io/devtools-protocol/tot/Performance/'.
func (e *CDPEngine) newScreenshotPagesTask(ctx context.Context, ss *[]*pageScreenshot, p url.URL, o *utils.ShareOptions) chromedp.Action {
	logger := utils.WithContext(ctx, e.logger)

	navigate := chromedp.ActionFunc(func(ctx context.Context) error {
		u, err := url.PathUnescape(p.String())
		if err != nil {
			return err
		}

		return withTimeout(chromedp.Navigate(u), e.config.MinActionTimeout).Do(ctx)
	})

	waitPage := withTimeout(waitLoaded(), e.config.MinActionTimeout)

	initialize := chromedp.ActionFunc(func(ctx context.Context) error {
		startedAt := time.Now().UTC()

		res := []byte(nil)
		initializeJS := "window.reportRenderer.initialize();"
		err := chromedp.Evaluate(initializeJS, &res).Do(ctx)
		if err != nil {
			logger.Error("couldn't initialize", zap.Error(err))

			return err
		}

		completedIn := time.Now().UTC().Sub(startedAt)
		logger.Info("initialized", zap.Duration("completedIn", completedIn))

		return nil
	})

	configure := chromedp.ActionFunc(func(ctx context.Context) error {
		startedAt := time.Now().UTC()

		loadReport := newReportLoadConfiguration(o)
		configJSON, err := json.Marshal(loadReport)
		if err != nil {
			logger.Error("couldn't marshal configuration", zap.Error(err))

			return err
		}

		res := []byte(nil)
		addConfigJS := fmt.Sprintf("window.reportRenderer.addConfig(%v);", string(configJSON))
		err = chromedp.Evaluate(addConfigJS, &res).Do(ctx)
		if err != nil {
			logger.Error("couldn't add configuration", zap.Error(err))

			return err
		}

		completedIn := time.Now().UTC().Sub(startedAt)
		logger.Info("added configuration", zap.Duration("completedIn", completedIn))

		return nil
	})

	loadReport := chromedp.ActionFunc(func(ctx context.Context) error {
		startedAt := time.Now().UTC()

		res := []byte(nil)
		exc := []byte(nil)
		loadReportJS := "window.reportRenderer.loadReport();"
		err := tryEvaluate(&res, &exc, loadReportJS, chromedp.EvalAsValue, evalAwait).Do(ctx)
		if err != nil {
			details, ok := err.(*runtime.ExceptionDetails)
			if ok && details.Exception.Type == runtime.TypeObject && details.Exception.Subtype == "" {
				loadingError := pbiError{}
				err2 := json.Unmarshal(exc, &loadingError)
				if err2 != nil {
					logger.Error("couldn't unmarshal error", zap.Error(err2))

					return err2
				}

				logger.Error("couldn't load report", zap.Error(&loadingError))

				return &loadingError
			}

			logger.Error("couldn't load report", zap.Error(err))

			return err
		}

		completedIn := time.Now().UTC().Sub(startedAt)
		logger.Info("loaded report", zap.Duration("completedIn", completedIn))
		return nil
	})

	takeScreenshots := chromedp.ActionFunc(func(ctx context.Context) error {
		startedAt := time.Now().UTC()

		for _, reportPage := range o.Pages {
			logger := logger.With(zap.String("pageID", reportPage.ID))

			pageIDJSON, err := json.Marshal(reportPage.ID)
			if err != nil {
				logger.Error("couldn't marshal page id", zap.Error(err))

				return err
			}

			res := []byte(nil)
			setPageJS := fmt.Sprintf("window.reportRenderer.setPage(%v);", string(pageIDJSON))
			err = chromedp.Evaluate(setPageJS, &res, evalAwait).Do(ctx)
			if err != nil {
				logger.Error("couldn't set page", zap.Error(err))

				return err
			}

			logger.Debug("navigated to page")

			pageSize := customPageSize{}
			getPageSizeJS := "window.reportRenderer.getPageSize();"
			err = chromedp.Evaluate(getPageSizeJS, &pageSize, chromedp.EvalAsValue).Do(ctx)
			if err != nil {
				logger.Error("couldn't get page size", zap.Error(err))

				return err
			}

			height := e.config.DefaultViewportHeight
			if pageSize.Height != 0 {
				height = pageSize.Height
			}

			height = height + e.config.ViewportMargin

			width := e.config.DefaultViewportWidth
			if pageSize.Width != 0 {
				width = pageSize.Width
			}

			width = width + e.config.ViewportMargin

			err = emulation.SetDeviceMetricsOverride(width, height, e.config.DisplayDensity, false).Do(ctx)
			if err != nil {
				logger.Error("couldn't set page size", zap.Error(err))

				return err
			}

			logger.Debug("set viewport size",
				zap.Int64("width", pageSize.Width),
				zap.Int64("height", pageSize.Height))

			exc := []byte(nil)
			startedAt := time.Now().UTC()
			renderReportJS := "window.reportRenderer.renderReport();"
			err = tryEvaluate(&res, &exc, renderReportJS, chromedp.EvalAsValue, evalAwait).Do(ctx)
			if err != nil {
				details, ok := err.(*runtime.ExceptionDetails)
				if ok && details.Exception.Type == runtime.TypeObject && details.Exception.Subtype == "" {
					renderingError := pbiError{}
					err2 := json.Unmarshal(exc, &renderingError)
					if err2 != nil {
						logger.Error("couldn't unmarshal error", zap.Error(err2))

						return err2
					}

					logger.Error("couldn't render report", zap.Error(&renderingError))

					return &renderingError
				}

				logger.Error("couldn't render report", zap.Error(err))

				return err
			}

			completedIn := time.Now().UTC().Sub(startedAt)
			logger.Debug("rendered page", zap.ByteString("res", res), zap.Duration("completedIn", completedIn))

			// NOTE: We have to wait for Bing maps visual to fully load since it doesn't respect report rendering completion event.
			time.Sleep(e.config.ScreenshotDelay)

			screenshot := pageScreenshot{
				pageID:   reportPage.ID,
				pageName: reportPage.Name,
			}
			err = chromedp.CaptureScreenshot(&screenshot.rawData).Do(ctx)
			if err != nil {
				logger.Error("couldn't capture screenshot", zap.Error(err))

				return err
			}

			logger.Debug("captured screenshot")

			*ss = append(*ss, &screenshot)
		}

		completedIn := time.Now().UTC().Sub(startedAt)
		logger.Info("rendered report", zap.Duration("completedIn", completedIn), zap.Int("totalPages", len(o.Pages)))

		return nil
	})

	return chromedp.Tasks{
		navigate,
		waitPage,
		initialize,
		configure,
		loadReport,
		takeScreenshots,
	}
}
