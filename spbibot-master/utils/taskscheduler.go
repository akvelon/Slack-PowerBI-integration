package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/cronexpr"
	"go.uber.org/zap"
)

// DefaultTaskScheduler is the global TaskScheduler instance.
var DefaultTaskScheduler TaskScheduler = newTaskScheduler()

// Schedule represents a Task execution plan.
type Schedule interface {
	fmt.Stringer
	nextAt(t time.Time) (time.Time, error)
}

// TODO: `github.com/hashicorp/cronexpr' doesn't seem to be actively maintained; we should use `github.com/robfig/cron/v3' when they implement extended specifiers we rely on.
type cronSchedule struct {
	timezone      string
	expression    *cronexpr.Expression
	rawExpression string
}

// NewSchedule creates a Schedule.
func NewSchedule(timezone, cronExpression string) (Schedule, error) {
	e, err := cronexpr.Parse(cronExpression)
	if err != nil {
		return nil, err
	}

	return &cronSchedule{
		timezone:      timezone,
		expression:    e,
		rawExpression: cronExpression,
	}, nil
}

func (s *cronSchedule) String() string {
	return s.rawExpression
}

func (s *cronSchedule) nextAt(t time.Time) (time.Time, error) {
	l, err := time.LoadLocation(s.timezone)
	if err != nil {
		return time.Time{}, err
	}

	lt := t.In(l)
	next := s.expression.Next(lt)
	if next.IsZero() {
		return time.Time{}, fmt.Errorf("no matching time")
	}

	return next.UTC(), nil
}

// Task represents a periodic task.
type Task struct {
	ID       int64
	Schedule Schedule
	Action   func(ctx context.Context) error
}

// TaskScheduler schedules & executes Task.
type TaskScheduler interface {
	Schedule(ctx context.Context, t *Task) error
}

type taskScheduler struct{}

func newTaskScheduler() TaskScheduler {
	return &taskScheduler{}
}

func (s *taskScheduler) Schedule(ctx context.Context, t *Task) error {
	l := WithContext(ctx, zap.L())
	l.Debug("scheduling task")

	ticker := newScheduledTicker(t.Schedule)

	SafeRoutine(func() {
		for {
			select {
			case <-ticker.C():
				l.Info("executing task")
				err := t.Action(ctx)
				if err != nil {
					l.Error("couldn't schedule task", zap.Error(err))
				}

			case <-ctx.Done():
				l.Info("stopping task")

				return
			}
		}
	})

	SafeRoutine(func() {
		ticker.Start(ctx)
	})

	return nil
}

// ticker2 is a Schedule & context.Context -aware alternative to time.Ticker.
type ticker2 interface {
	C() <-chan time.Time
	Start(ctx context.Context)
	nextTickIn(after time.Time) (time.Duration, error)
}

type scheduledTicker struct {
	c chan time.Time
	s Schedule
}

func newScheduledTicker(s Schedule) ticker2 {
	return &scheduledTicker{
		c: make(chan time.Time, 1),
		s: s,
	}
}

func (t *scheduledTicker) C() <-chan time.Time {
	return t.c
}

func (t *scheduledTicker) Start(ctx context.Context) {
	l := WithContext(ctx, zap.L()).
		With(zap.String("schedule", t.s.String()))

	current := time.Now().UTC()
	for {
		nextIn, err := t.nextTickIn(current)
		if err != nil {
			break
		}

		l.Debug("starting ticker", zap.Time("current", current), zap.Duration("nextIn", nextIn))
		next := time.NewTimer(nextIn)
		select {
		case current = <-next.C:
			l.Debug("next tick", zap.Time("current", current))
			t.c <- current
			next.Stop()

		case <-ctx.Done():
			l.Debug("stopping ticker")
			next.Stop()

			return
		}
	}
}

func (t *scheduledTicker) nextTickIn(after time.Time) (time.Duration, error) {
	nextAt, err := t.s.nextAt(after)
	if err != nil {
		return 0, err
	}

	return nextAt.Sub(after), nil
}
