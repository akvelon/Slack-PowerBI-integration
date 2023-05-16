package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/julienschmidt/httprouter"
	"github.com/replaygaming/amplitude"
	"go.uber.org/zap"


)

func main() {
	fallbackLogger := log.New(os.Stderr, "ERROR ", log.Ldate|log.Ltime|log.Lshortfile|log.LUTC|log.Lmsgprefix)

	baseProvider, err := config.NewDotenvProvider("./env/base.env")
	if err != nil {
		fallbackLogger.Println("couldn't create config provider:", err)

		return
	}

	botProvider, err := config.NewDotenvProvider("bot.env")
	if err != nil {
		fallbackLogger.Println("couldn't create config provider:", err)

		return
	}

	configProvider := config.NewProviderChain(botProvider, baseProvider)
	conf, err := config.NewBotConfig(configProvider)
	if err != nil {
		fallbackLogger.Println("couldn't create config:", err)

		return
	}

	awsSession, err := aws2.NewSessionBuilder().
		WithAWSConfig(conf.AWS).
		WithStdLogger(fallbackLogger).
		NewSession()
	if err != nil {
		fallbackLogger.Println("couldn't create AWS session:", err)

		return
	}

	cloudWatchLogs := cloudwatchlogs.New(awsSession)
	logger, syncLogger, err := logging.NewBuilder().
		WithHostConfig(conf.Host).
		WithLoggerConfig(conf.Logger).
		WithFallbackLogger(fallbackLogger).
		WithCloudWatchLogs(cloudWatchLogs).
		NewLogger()
	if err != nil {
		fallbackLogger.Println("couldn't create logger:", err)

		return
	}

	defer syncLogger()

	pid := os.Getpid()
	logger = logger.With(zap.Int("pid", pid))

	hostname, err := os.Hostname()
	if err != nil {
		logger.Error("couldn't get hostname", zap.Error(err))
	} else {
		logger = logger.With(zap.String("host", hostname))
	}

	logger.Info("starting")

	mysqlConn, err := db.InitDB("mysql", conf.DB)
	if err != nil {
		logger.Error("couldn't connect to DB", zap.Error(err))

		return
	}

	defer func() {
		logger.Debug("closing DB connection")
		err := mysqlConn.Close()
		if err != nil {
			logger.Error("couldn't close DB connection", zap.Error(err))
		}
	}()

	cdpEngine := cdpengine.NewCDPEngine(conf.Browser, logger)
	err = cdpEngine.Start(context.Background())
	if err != nil {
		logger.Error("couldn't start Chrome instance", zap.Error(err))

		return
	}

	defer func() {
		logger.Debug("stopping Chrome instance")
		err := cdpEngine.Stop()
		if err != nil && err != context.Canceled {
			logger.Error("couldn't stop Chrome instance", zap.Error(err))
		}
	}()

	mysqlUserRepository := mysqlDB.NewMysqlUserRepository(mysqlConn, logger)
	mysqlUserTokenRepository := mysqlDB.NewMysqlUserTokenRepository(mysqlUserRepository, logger)
	mysqlWorkspaceRepository := mysqlDB.NewMysqlWorkspaceRepository(mysqlConn, logger)
	mysqlAlertRepository := mysqlDB.NewMysqlAlertRepository(mysqlConn, logger)
	mysqlFilterRepository := mysqlDB.NewMySQLFilterRepository(mysqlConn, logger)
	mysqlPostingTaskRepository := mysqlDB.NewMySQLPostReportTaskRepository(mysqlConn, logger)

	powerBiClient := powerbi.NewServiceClient(conf.OAuthConfig, &conf.PowerBiClient, mysqlUserTokenRepository, logger)

	dbQueryTimeout := time.Duration(conf.DB.Timeout) * time.Second

	mq := messagequeue.MessageQueue(nil)
	switch conf.MessageQueue.Implementation {
	case config.MQSQS:
		q := sqs.New(awsSession)
		mq = messagequeue.NewSQSMessageQueue(q, conf.MessageQueue, logger)

	case config.MQInProcess:
		mq = messagequeue.NewInProcessMessageQueue()

	default:
		logger.Error("unknown message queue implementation", zap.String("implementation", string(conf.MessageQueue.Implementation)))

		return
	}

	botErrorHandler := useCase.NewBotErrorHandler(logger)
	schedulerErrorHandler := useCase.NewSchedulerErrorHandler(mysqlPostingTaskRepository, logger, powerBiClient)
	deletedChannelsHandler := useCase.NewDeletedChannelsHandler(mysqlPostingTaskRepository, mysqlWorkspaceRepository, logger)
	activePagesFilter := useCase.NewActivePagesFilter(*powerBiClient, schedulerErrorHandler, mysqlWorkspaceRepository, logger, mysqlPostingTaskRepository)
	userUsecase := useCase.NewUserUsecase(mysqlUserRepository, dbQueryTimeout, conf.DB.UserIDHashCost, conf.OAuthConfig, logger)
	reportUsecase := useCase.NewReportUsecase(*powerBiClient, mysqlWorkspaceRepository, mysqlPostingTaskRepository, mysqlUserRepository, mq, dbQueryTimeout, logger, conf.FeatureToggles, botErrorHandler, schedulerErrorHandler, activePagesFilter, deletedChannelsHandler)
	workspaceUsecase := useCase.NewWorkspaceUsecase(mysqlWorkspaceRepository, dbQueryTimeout)
	alertUsecase := useCase.NewAlertUsecase(mysqlAlertRepository, *powerBiClient, mysqlUserTokenRepository, mysqlWorkspaceRepository, dbQueryTimeout, logger, botErrorHandler)
	filterUsecase := useCase.NewFilterUsecase(mysqlFilterRepository, dbQueryTimeout)

	alertUsecase.ScheduleAlertsCheck(context.Background()) // schedule check alerts tasks

	if conf.FeatureToggles.ReportScheduling {
		scheduleTasksCtx, cancelScheduling := context.WithCancel(context.Background())
		defer cancelScheduling()
		utils.SafeRoutine(func() {
			reportUsecase.StartScheduledPosting(scheduleTasksCtx)
		})
	}

	analytics.SetDefaultAmplitudeClient(amplitude.NewClient(conf.AmplitudeKey), logger)

	switch conf.MessageQueue.Implementation {
	case config.MQSQS:

	default:
		logger.Error("unknown message queue implementation", zap.String("implementation", string(conf.MessageQueue.Implementation)))

		return
	}

	router := httprouter.New()
	router.PanicHandler = newPanicHandler(logger)

	httpHandler.NewSlashCommandHandler(router, reportUsecase, userUsecase, workspaceUsecase, alertUsecase, conf.Slack, &conf.OAuthConfig, conf.FeatureToggles, logger)
	httpHandler.NewInteractionPayloadHandler(router, reportUsecase, userUsecase, workspaceUsecase, alertUsecase, filterUsecase, mq, conf.Slack, &conf.OAuthConfig, conf.FeatureToggles, logger)
	httpHandler.NewBotAuthHandler(router, workspaceUsecase, conf.BotAccessTokenConfig, logger)
	httpHandler.NewEventsHandler(router, userUsecase, workspaceUsecase, conf.Slack, &conf.OAuthConfig, conf.FeatureToggles, logger)
	httpHandler.ConfigureStaticFilesHandler(router)
	httpHandler.ConfigureHealthCheck(router)

	if conf.TestAPI.Enable {
		httpHandler.ConfigureTestAPIHandler(router, reportUsecase, mq, conf.TestAPI, logger)
	}

	pipeline := middlewares.NewRouterMiddleware(router)
	pipeline = middlewares.NewCORSMiddleware(pipeline)
	pipeline = middlewares.NewRequestLoggingMiddleware(pipeline, conf.RequestLogging, logger)
	pipeline = middlewares.NewRequestIDMiddleware(pipeline)

	httpAddress := utils.JoinHostPort("", conf.Server.Port)
	httpsAddress := utils.JoinHostPort("", conf.Server.TLSPort)

	shutdownRequested := make(chan os.Signal, 1)
	go signal.Notify(shutdownRequested, os.Interrupt, syscall.SIGTERM)

	switch conf.Host.Environment {
	case config.EnvironmentDevelopment:
		server := http.Server{
			Addr:    httpAddress,
			Handler: pipeline,
		}

		go func() {
			err = server.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				logger.Error("server error", zap.Error(err), zap.String("address", server.Addr))
			}
		}()

		defer func() {
			logger.Debug("stopping HTTP server")
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(conf.Host.ShutdownTimeout)*time.Second)
			defer cancel()
			err := server.Shutdown(ctx)
			if err != nil {
				logger.Warn("shut down w/ errors", zap.Error(err))
			}

			browser.Dispose()
		}()

	case config.EnvironmentProduction:
		httpPipeline := middlewares.NewHTTPSRedirectionMiddleware(pipeline, conf.Server.TLSPort)
		httpServer := http.Server{
			Addr:    httpAddress,
			Handler: httpPipeline,
		}

		go func() {
			err = httpServer.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				logger.Error("server error", zap.Error(err), zap.String("address", httpServer.Addr))
			}
		}()

		httpsServer := http.Server{
			Addr:    httpsAddress,
			Handler: pipeline,
		}

		go func() {
			err = httpsServer.ListenAndServeTLS(conf.Server.Certificate, conf.Server.Key)
			if err != nil && err != http.ErrServerClosed {
				logger.Error("server error", zap.Error(err), zap.String("address", httpsServer.Addr))
			}
		}()

		defer func() {
			done := sync.WaitGroup{}
			for _, s := range []*http.Server{&httpServer, &httpsServer} {
				done.Add(1)
				go func(s *http.Server) {
					logger.Debug("stopping HTTP server")
					ctx, cancel := context.WithTimeout(context.Background(), time.Duration(conf.Host.ShutdownTimeout)*time.Second)
					defer cancel()
					err := s.Shutdown(ctx)
					if err != nil {
						logger.Warn("shut down w/ errors", zap.Error(err))
					}

					done.Done()
				}(s)
			}

			done.Wait()
		}()

	default:
		logger.Error("unknown environment", zap.String("environment", string(conf.Host.Environment)))

		return
	}

	logger.Info("ready")

	s := <-shutdownRequested
	logger.Info("shutting down", zap.String("signal", s.String()))
}

func newPanicHandler(l *zap.Logger) func(http.ResponseWriter, *http.Request, interface{}) {
	return func(w http.ResponseWriter, r *http.Request, p interface{}) {
		l := utils.WithContext(r.Context(), l)
		l.Error("route handler panicked", zap.Any("panic", p))
		w.WriteHeader(http.StatusInternalServerError)
	}
}
