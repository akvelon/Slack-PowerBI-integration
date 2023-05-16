package browser

import (
	"context"

	"github.com/chromedp/chromedp"
	"go.uber.org/zap"
)

type browser struct {
	browserContext   context.Context
	closeBrowserFunc context.CancelFunc
}

// Browser provides a interface to browser functions
type Browser interface {
	GetContext() *context.Context
}

var brInstance *browser

func (b *browser) GetContext() *context.Context {
	return &b.browserContext
}

// GetBrowserInstance returns a screenshot singleton
func GetBrowserInstance() (Browser, error) {
	var err error
	if brInstance == nil {
		brInstance = new(browser)
		brInstance.browserContext, brInstance.closeBrowserFunc = chromedp.NewContext(context.Background())
		err = chromedp.Run(brInstance.browserContext)
	}

	return brInstance, err
}

// Dispose releases browser instance
func Dispose() {
	if brInstance != nil && brInstance.browserContext != nil {
		zap.L().Info("###DisposeBrowserInstance")
		brInstance.closeBrowserFunc()
	}
}
