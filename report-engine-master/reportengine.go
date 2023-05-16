package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

)

func main() {
	fallbackLogger := log.New(os.Stderr, "ERROR ", log.Ldate|log.Ltime|log.Lshortfile|log.LUTC|log.Lmsgprefix)

	baseProvider, err := config.NewDotenvProvider("./env/base.env")
	if err != nil {
		fallbackLogger.Println("couldn't create config provider:", err)

		return
	}

	reportEngineProvider, err := config.NewDotenvProvider("reportengine.env")
	if err != nil {
		fallbackLogger.Println("couldn't create config provider:", err)

		return
	}

	configProvider := config.NewProviderChain(reportEngineProvider, baseProvider)
	conf, err := config.NewReportEngineConfig(configProvider)
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

	mysqlUserRepository := mysqlDB.NewMysqlUserRepository(mysqlConn, logger)
	mysqlUserTokenRepository := mysqlDB.NewMysqlUserTokenRepository(mysqlUserRepository, logger)
	mysqlWorkspaceRepository := mysqlDB.NewMysqlWorkspaceRepository(mysqlConn, logger)
	mysqlPostingTaskRepository := mysqlDB.NewMySQLPostReportTaskRepository(mysqlConn, logger)

	powerBiClient := powerbi.NewServiceClient(conf.OAuthConfig, &conf.PowerBiClient, mysqlUserTokenRepository, logger)

	dbQueryTimeout := time.Duration(conf.DB.Timeout) * time.Second

	mq := messagequeue.MessageQueue(nil)
	switch conf.MessageQueue.Implementation {
	case config.MQSQS:
		q := sqs.New(awsSession)
		mq = messagequeue.NewSQSMessageQueue(q, conf.MessageQueue, logger)

	default:
		logger.Error("unknown message queue implementation", zap.String("implementation", string(conf.MessageQueue.Implementation)))

		return
	}

	cdpEngine := reportengine.NewCDPReportEngine(conf.Browser, logger)
	reportengine.SetDefaultReportEngine(cdpEngine)
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

	retryStrategy := reportengine.NewRetryStrategy(logger, conf.MessageQueue.URL, conf.MaxAttempts, awsSession)
	userUsecase := useCase.NewUserUsecase(mysqlUserRepository, dbQueryTimeout, conf.DB.UserIDHashCost, conf.OAuthConfig, logger)
	reportUsecase := useCase.NewReportUsecase(*powerBiClient, mysqlWorkspaceRepository, mysqlPostingTaskRepository, mysqlUserRepository, mq, dbQueryTimeout, logger, retryStrategy)
	workspaceUsecase := useCase.NewWorkspaceUsecase(mysqlWorkspaceRepository, dbQueryTimeout)

	analytics.SetDefaultAmplitudeClient(amplitude.NewClient(conf.AmplitudeKey), logger)

	handleMessagesCtx, cancelHandling := context.WithCancel(context.Background())
	dispatcher := messageHandler.NewMessageDispatcher(mq, conf.MessageHandler, logger)
	handleReportMessages := messageHandler.NewReportWorker(reportUsecase, userUsecase, workspaceUsecase, logger)
	err = dispatcher.RegisterWorker(handleReportMessages)
	if err != nil {
		logger.Error("couldn't register worker", zap.Error(err))

		return
	}

	dispatcher.Start(handleMessagesCtx)

	defer func() {
		logger.Debug("stopping message handling")
		stopHandlingCtx, cancelStopping := context.WithTimeout(handleMessagesCtx, time.Duration(conf.Host.ShutdownTimeout)*time.Second)
		defer cancelStopping()
		err := dispatcher.Stop(stopHandlingCtx)
		if err != nil && err != context.Canceled {
			logger.Error("couldn't stop message handling", zap.Error(err))
		}
	}()

	defer cancelHandling()

	logger.Info("ready")

	shutdownRequested := make(chan os.Signal, 1)
	go signal.Notify(shutdownRequested, os.Interrupt, syscall.SIGTERM)
	s := <-shutdownRequested
	logger.Info("shutting down", zap.String("signal", s.String()))
}
