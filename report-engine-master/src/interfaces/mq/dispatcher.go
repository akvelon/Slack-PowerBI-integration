package mq

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"go.uber.org/zap"

	
)

type registryEntry struct {
	w            Worker
	subscription chan *messagequeue.Envelope
}

type messageDispatcher struct {
	registry      map[messagequeue.MessageKind]*registryEntry
	waitWorkers   sync.WaitGroup
	workCompleted chan struct{}
	mq            messagequeue.MessageQueue
	config        *config.MessageHandlerConfig
	logger        *zap.Logger
}

// NewMessageDispatcher creates a messagequeue.MessageQueue -backed Manager.
func NewMessageDispatcher(
	m messagequeue.MessageQueue,
	c *config.MessageHandlerConfig,
	l *zap.Logger,
) Manager {
	return &messageDispatcher{
		registry:      map[messagequeue.MessageKind]*registryEntry{},
		waitWorkers:   sync.WaitGroup{},
		workCompleted: make(chan struct{}),
		mq:            m,
		config:        c,
		logger:        l,
	}
}

func (d *messageDispatcher) RegisterWorker(w Worker) error {
	for _, k := range w.SupportedMessages() {
		_, ok := d.registry[k]
		if ok {
			return fmt.Errorf("a worker is already registered for %v", k)
		}

		e := registryEntry{
			w:            w,
			subscription: make(chan *messagequeue.Envelope),
		}
		d.registry[k] = &e
	}

	return nil
}

func (d *messageDispatcher) Start(ctx context.Context) {
	utils.SafeRoutine(func() {
		d.waitWorkers.Wait()
		d.workCompleted <- struct{}{}
	})

	utils.SafeRoutine(func() {
		logger := utils.WithContext(ctx, d.logger)
		for {
			select {
			case <-ctx.Done():
				return

			default:
				break
			}

			e, err := d.mq.Peek(ctx, messagequeue.Wait)
			if err != nil {
				if err != messagequeue.ErrNoMessages {
					logger.Debug("couldn't receive message", zap.Error(err))
				}

				continue
			}

			ctx = utils.WithActivityInfo(ctx, map[string]string{
				"activityID":  e.TraceID,
				"messageID":   e.ID,
				"messageKind": string(e.Kind),
			})
			l := utils.WithContext(ctx, logger)

			w, ok := d.registry[e.Kind]
			if !ok {
				l.Error("no suitable worker")
				err = d.mq.Delete(ctx, e.Handle)
				if err != nil {
					l.Error("couldn't delete message", zap.Error(err))
				}

				continue
			}

			w.subscription <- e
		}
	})

	for k, w := range d.registry {
		for i := 0; i < int(d.config.ConcurrencyLevel); i++ {
			d.spawnWorker(ctx, w.w, w.subscription, k, i)
		}
	}
}

func (d *messageDispatcher) Stop(ctx context.Context) error {
	for {
		select {
		case <-d.workCompleted:
			return nil

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (d *messageDispatcher) spawnWorker(ctx context.Context, w Worker, subscription <-chan *messagequeue.Envelope, k messagequeue.MessageKind, id int) {
	d.waitWorkers.Add(1)
	utils.SafeRoutine(func() {
		defer d.waitWorkers.Done()

		ctx = utils.WithActivityInfo(ctx, map[string]string{
			"messageKind": string(k),
			"workerID":    strconv.FormatInt(int64(id), 10),
		})
		logger := utils.WithContext(ctx, d.logger)
		logger.Debug("started")

		for {
			select {
			case <-ctx.Done():
				return

			default:
				break
			}

			e := <-subscription

			ctx = utils.WithActivityInfo(ctx, map[string]string{
				"activityID": e.TraceID,
				"messageID":  e.ID,
			})
			l := utils.WithContext(ctx, logger)
			l.Debug("received message")

			err := d.mq.Delete(ctx, e.Handle)
			if err != nil {
				l.Error("couldn't delete message", zap.Error(err))
			}

			err = w.Handle(ctx, e)
			if err != nil {
				l.Error("couldn't handle message", zap.Error(err))
			}
		}
	})
}
