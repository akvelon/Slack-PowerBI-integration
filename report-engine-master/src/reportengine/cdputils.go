package reportengine

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"

	
)

type zapCDPAdapter struct {
	*zap.Logger
}

func newZapCDPAdapter(l *zap.Logger) *zapCDPAdapter {
	return &zapCDPAdapter{
		Logger: l,
	}
}

func (l *zapCDPAdapter) Log(f string, vs ...interface{}) {
	l.Sugar().Debugf(f, vs)
}

func noHeadless(a *chromedp.ExecAllocator) {
	chromedp.Flag("headless", false)(a)
	chromedp.Flag("hide-scrollbars", false)(a)
	chromedp.Flag("mute-audio", false)(a)
}

func evalTimeout(t time.Duration) chromedp.EvaluateOption {
	return func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
		return p.WithTimeout(runtime.TimeDelta(t.Milliseconds()))
	}
}

func withTimeout(a chromedp.Action, d time.Duration) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		taskCtx, cancelTask := context.WithTimeout(ctx, d)
		defer cancelTask()

		return a.Do(taskCtx)
	})
}

func withLogging(a chromedp.Action, actionTag string, l *zap.Logger) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		startedAt := time.Now().UTC()

		l.Debug("started " + actionTag)
		err := a.Do(ctx)

		completedIn := time.Now().UTC().Sub(startedAt)
		l.Debug("completed "+actionTag, zap.Duration("completedIn", completedIn))

		return err
	})
}

func executeJS(res interface{}, expression string, arg interface{}) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		js := expression
		if arg != nil {
			argJSON, err := json.Marshal(arg)
			if err != nil {
				return err
			}

			js = fmt.Sprintf(expression, string(argJSON))
		}

		return chromedp.
			Evaluate(js, &res, chromedp.EvalAsValue).
			Do(ctx)
	})
}

func evalAwait(p *runtime.EvaluateParams) *runtime.EvaluateParams {
	return p.WithAwaitPromise(true)
}

func waitLoaded() chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		err := page.Enable().Do(ctx)
		if err != nil {
			return err
		}

		err = page.SetLifecycleEventsEnabled(true).Do(ctx)
		if err != nil {
			return err
		}

		loadedCtx, cancelLoaded := context.WithCancel(ctx)
		loadedChan := make(chan struct{})

		chromedp.ListenTarget(loadedCtx, func(ev interface{}) {
			_, ok := ev.(*page.EventLifecycleEvent)
			if ok {
				cancelLoaded()
				close(loadedChan)
			}
		})

		select {
		case <-loadedChan:
			return nil

		case <-ctx.Done():
			return ctx.Err()
		}
	})
}

func timestamp(ctx context.Context) string {
	t := time.Time{}
	r := uint32(0)

	activityID := utils.ActivityInfo(ctx)["activityID"]
	if activityID != "" {
		id, err := ksuid.Parse(activityID)
		if err == nil && !id.IsNil() {
			t = id.Time().UTC()
			r = binary.BigEndian.Uint32(id.Payload())
		}
	}

	if t.IsZero() {
		t = time.Now().UTC()
		r = uint32(rand.Int31())
	}

	return fmt.Sprintf("%v %v", t.Format(time.RFC3339), strconv.FormatInt(int64(r), 10))
}

func tryEvaluate(res interface{}, exc interface{}, js string, os ...chromedp.EvaluateOption) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		p := runtime.Evaluate(js)
		for _, o := range os {
			p = o(p)
		}

		r, e, err := p.Do(ctx)
		if err != nil {
			return err
		}

		if e != nil {
			err := unmarshalRemoteObject(exc, r)
			if err != nil {
				return err
			}

			return e
		}

		return unmarshalRemoteObject(res, r)
	})
}

func unmarshalRemoteObject(res interface{}, r *runtime.RemoteObject) error {
	if res == nil {
		return fmt.Errorf("res cannot be nil")
	}

	switch res := res.(type) {
	case **runtime.RemoteObject:
		*res = r

	case *[]byte:
		*res = r.Value

	default:
		err := json.Unmarshal(r.Value, res)
		if err != nil {
			return err
		}
	}

	return nil
}
