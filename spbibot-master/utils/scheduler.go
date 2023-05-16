package utils

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

type singleTask struct {
	TimeTicker  *time.Ticker
	StopChannel *chan bool
}

type scheduler struct {
	PeriodicalTasks map[int64]*singleTask
}

// Scheduler provides features to add/update/kill periodical tasks
type Scheduler interface {
	AddTask(ctx context.Context, taskID int64, task func() error, interval time.Duration, onException func(context.Context, int64, error))
	KillTask(taskID int64) bool
}

var (
	instance     *scheduler
	once         sync.Once
	defaultTasks = make(map[int64]*singleTask)
)

// GetInstance returns a scheduler singleton
func GetInstance() Scheduler {
	once.Do(func() {
		instance = &scheduler{PeriodicalTasks: defaultTasks}
	})

	return instance
}

// AddTask creates new periodical task
func (s *scheduler) AddTask(ctx context.Context, taskID int64, task func() error, interval time.Duration, onException func(context.Context, int64, error)) {
	stop := make(chan bool)
	ticker := time.NewTicker(interval)

	go func() {
		l := WithContext(ctx, zap.L()).
			With(zap.Int64("taskID", taskID))

		err := task() // first tick
		if err != nil {
			l.Error("task stopped from error", zap.Error(err))
			s.KillTask(taskID)
			onException(ctx, taskID, err)

			return
		}

		for {
			select {
			case <-ticker.C:
				l.Info("task ticked")
				err := task()
				if err != nil {
					ticker.Stop()
					l.Error("task stopped from error", zap.Error(err))
					s.KillTask(taskID)
					onException(ctx, taskID, err)

					return
				}

			case <-stop:
				l.Info("task stopped manually")

				return
			}
		}
	}()

	t := singleTask{ticker, &stop}
	s.PeriodicalTasks[taskID] = &t
}

// KillTask kills a task by it's ID
func (s *scheduler) KillTask(taskID int64) bool {
	t, ok := s.PeriodicalTasks[taskID]
	if !ok {
		return ok
	}

	close(*t.StopChannel)
	t.TimeTicker.Stop()
	delete(s.PeriodicalTasks, taskID)
	return true
}
