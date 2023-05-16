package cdpengine

import (
	"context"

	"github.com/chromedp/chromedp"
	"go.uber.org/zap"


)

// CDPEngine is an engine powered by a Chrome instance controlled via Chrome DevTools Protocol.
type CDPEngine struct {
	config            *config.BrowserConfig
	logger            *zap.Logger
	allocatorOptions  []chromedp.ExecAllocatorOption
	allocatorCtx      context.Context
	browserCtxOptions []chromedp.ContextOption
	browserCtx        context.Context
}

// NewCDPEngine creates a Chrome-based engine.
func NewCDPEngine(c *config.BrowserConfig, l *zap.Logger) *CDPEngine {
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

func (e *CDPEngine) startBrowser() error {
	e.browserCtx, _ = chromedp.NewContext(e.allocatorCtx, e.browserCtxOptions...)

	return chromedp.Run(e.browserCtx)
}
