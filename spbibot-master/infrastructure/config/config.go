package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
	"golang.org/x/oauth2"

	
)

const (
	msHost        = "https://login.microsoftonline.com/"
	msAuthParam   = "/oauth2/authorize"
	msTokenParam  = "/oauth2/token"
	msLogoutParam = "/oauth2/logout"
	slackOauthURL = "https://slack.com/oauth/v2/authorize"
)

// Environment controls environment-dependent features.
type Environment string

const (
	// EnvironmentDevelopment enables development mode.
	EnvironmentDevelopment Environment = "development"
	// EnvironmentProduction enables production mode.
	EnvironmentProduction Environment = "production"
)

func parseEnvironment(s string) (Environment, error) {
	switch Environment(s) {
	case EnvironmentDevelopment, EnvironmentProduction:
		return Environment(s), nil

	default:
		return "", fmt.Errorf("unknown environment: %v", s)
	}
}

// HostConfig controls application-wide behavior.
type HostConfig struct {
	Environment     Environment `envconfig:"ENVIRONMENT"`
	ShutdownTimeout int         `envconfig:"SERVER_SHUTDOWNTIMEOUT"`
}

func newHostConfig(p Provider) (*HostConfig, error) {
	e, err := parseEnvironment(p.Get("ENV", string(EnvironmentDevelopment)))
	if err != nil {
		return nil, err
	}

	return &HostConfig{
		Environment:     e,
		ShutdownTimeout: getInt(p, "SERVER_SHUTDOWNTIMEOUT", 10),
	}, nil
}

// ServerConfig type defines server properties in a config
type ServerConfig struct {
	Port        int    `envconfig:"SERVER_PORT"`
	TLSPort     int    `envconfig:"SERVER_TLSPORT"`
	Host        string `envconfig:"SERVER_HOST"`
	Certificate string `envconfig:"SERVER_CERTIFICATE"`
	Key         string `envconfig:"SERVER_KEY"`
}

// DatabaseConfig type defines server properties in a config
type DatabaseConfig struct {
	Port           string `envconfig:"DB_PORT"`
	Host           string `envconfig:"DB_HOST"`
	Name           string `envconfig:"DB_NAME"`
	Username       string `envconfig:"DB_USERNAME"`
	UserPwd        string `envconfig:"DB_USERNAME_PWD"`
	Timeout        int    `envconfig:"TIMEOUT"`
	UserIDHashCost int    `envconfig:"USERID_HASH_COST"`
}

// BaseConfig controls common features.
type BaseConfig struct {
	Host                 *HostConfig
	DB                   *DatabaseConfig
	OAuthConfig          oauth.Config
	Slack                *SlackConfig
	PowerBiClient        PowerBiClientConfig
	BotAccessTokenConfig oauth.Config
	Logger               *LoggerConfig
	FeatureToggles       *FeatureTogglesConfig
	Browser              *BrowserConfig
	AmplitudeKey         string `envconfig:"AMPLITUDE_API_KEY"`
	MessageQueue         *MessageQueueConfig
	MessageHandler       *MessageHandlerConfig
	AWS                  *AWSConfig
}

// BotConfig controls cmd/bot behavior.
type BotConfig struct {
	*BaseConfig
	Server         ServerConfig
	RequestLogging *RequestLoggingConfig
	TestAPI        *TestAPIConfig
}

// SlackConfig controls interaction w/ Slack.
type SlackConfig struct {
	SigningSecret  string `envconfig:"SLACK_SIGN_IN_SECRET"`
	VerifyRequests bool   `envconfig:"SLACK_VERIFYREQUESTS"`
}

func newSlackConfig(p Provider) (*SlackConfig, error) {
	const prefix = "SLACK"

	v := getBool(p, prefix+"_VERIFYREQUESTS", true)

	s := p.Get(prefix+"_SIGN_IN_SECRET", "")
	if s == "" && v {
		return nil, fmt.Errorf("either signing secret must be set or request verification disabled")
	}

	return &SlackConfig{
		SigningSecret:  s,
		VerifyRequests: v,
	}, nil
}

// PowerBiClientConfig defines runtime configuration for interaction w/ Power BI REST API.
type PowerBiClientConfig struct {
	APIURL       string `envconfig:"POWERBICLIENT_APIURL"`
	ClientID     string `envconfig:"CLIENT_ID"`
	ClientSecret string `envconfig:"CLIENT_SECRET"`
	Endpoint     oauth2.Endpoint
}

// LoggerConfig controls logger behavior.
type LoggerConfig struct {
	Level        zapcore.Level        `envconfig:"LOGGER_LEVEL"`
	Encoding     string               `envconfig:"LOGGER_ENCODING"`
	LevelEncoder zapcore.LevelEncoder `envconfig:"LOGGER_LEVELENCODER"`
	Sinks        []string             `envconfig:"LOGGER_SINKS"`
	ErrorSinks   []string             `envconfig:"LOGGER_ERRORSINKS"`
	MaxSizeMB    int                  `envconfig:"LOGGER_MAXSIZEMB"`
	MaxAgeDays   int                  `envconfig:"LOGGER_MAXAGEDAYS"`
	MaxBackups   int                  `envconfig:"LOGGER_MAXBACKUPS"`
	BatchSize    uint                 `envconfig:"LOGGER_BATCHSIZE"`
}

func newLoggerConfig(p Provider) (*LoggerConfig, error) {
	const prefix = "LOGGER"

	var l zapcore.Level
	err := l.UnmarshalText([]byte(p.Get(prefix+"_LEVEL", "info")))
	if err != nil {
		return nil, err
	}

	var le zapcore.LevelEncoder
	err = le.UnmarshalText([]byte(p.Get(prefix+"_LEVELENCODER", "capitalColor")))
	if err != nil {
		return nil, err
	}

	f := getBool(p, prefix+"_ENABLEFILE", true)
	s := getBool(p, prefix+"_ENABLESTDOUT", true)
	if !f && !s {
		return nil, fmt.Errorf("at least one sink must be enabled")
	}

	ss := []string(nil)
	err = json.Unmarshal([]byte(p.Get(prefix+"_SINKS", `["stdout"]`)), &ss)
	if err != nil {
		return nil, errors.Wrap(err, "no sinks set")
	}

	ess := []string(nil)
	err = json.Unmarshal([]byte(p.Get(prefix+"_ERRORSINKS", `["stderr"]`)), &ess)
	if err != nil {
		return nil, errors.Wrap(err, "no error sinks set")
	}

	return &LoggerConfig{
		Level:        l,
		Encoding:     p.Get("LOGGER_ENCODING", "console"),
		LevelEncoder: le,
		Sinks:        ss,
		ErrorSinks:   ess,
		MaxSizeMB:    getInt(p, "LOGGER_MAXSIZEMB", 128),
		MaxAgeDays:   getInt(p, "LOGGER_MAXAGEDAYS", 168),
		MaxBackups:   getInt(p, "LOGGER_MAXBACKUPS", 16),
		BatchSize:    getUint(p, "LOGGER_BATCHSIZE", 2),
	}, nil
}

// FeatureTogglesConfig holds feature toggles.
type FeatureTogglesConfig struct {
	ReportScheduling       bool `envconfig:"FEATURETOGGLES_REPORTSCHEDULING"`
	DeletedChannelsHandler bool `envconfig:"FEATURETOGGLES_DELETEDCHANNELS_HANDLER"`
	PaymentIntroduction    bool `envconfig:"FEATURETOGGLES_PAYMENT_INTRODUCTION"`
}

func newFeatureTogglesConfig(p Provider) *FeatureTogglesConfig {
	const prefix = "FEATURETOGGLES"

	return &FeatureTogglesConfig{
		ReportScheduling:       getBool(p, prefix+"_REPORTSCHEDULING", false),
		DeletedChannelsHandler: getBool(p, prefix+"_DELETEDCHANNELS_HANDLER", false),
		PaymentIntroduction:    getBool(p, prefix+"_PAYMENT_INTRODUCTION", false),
	}
}

// BrowserConfig controls browser behavior.
type BrowserConfig struct {
	Headless              bool          `envconfig:"BROWSER_HEADLESS"`
	RedirectLog           bool          `envconfig:"BROWSER_REDIRECTLOG"`
	TabTimeout            time.Duration `envconfig:"BROWSER_TABTIMEOUT"`
	MinActionTimeout      time.Duration `envconfig:"BROWSER_MINACTIONTIMEOUT"`
	DefaultViewportHeight int64         `envconfig:"BROWSER_DEFAULTVIEWPORTHEIGHT"`
	DefaultViewportWidth  int64         `envconfig:"BROWSER_DEFAULTVIEWPORTWIDTH"`
	ViewportMargin        int64         `envconfig:"BROWSER_VIEWPORTMARGIN"`
	DisplayDensity        float64       `envconfig:"BROWSER_DISPLAYDENSITY"`
	ResourcesDirectory    string        `envconfig:"BROWSER_RESOURCESDIRECTORY"`
	ScreenshotDelay       time.Duration `envconfig:"BROWSER_SCREENSHOTDELAY"`
}

func newBrowserConfig(p Provider) (*BrowserConfig, error) {
	const prefix = "BROWSER"

	minActionTimeout, err := time.ParseDuration(p.Get(prefix+"_MINACTIONTIMEOUT", "30s"))
	if err != nil {
		return nil, err
	}

	tabTimeout, err := time.ParseDuration(p.Get(prefix+"_TABTIMEOUT", "15m"))
	if err != nil {
		return nil, err
	}

	screenshotDelay, err := time.ParseDuration(p.Get(prefix+"_SCREENSHOTDELAY", "3s"))
	if err != nil {
		return nil, err
	}

	return &BrowserConfig{
		Headless:              getBool(p, prefix+"_HEADLESS", true),
		RedirectLog:           getBool(p, prefix+"_REDIRECTLOG", false),
		TabTimeout:            tabTimeout,
		MinActionTimeout:      minActionTimeout,
		DefaultViewportHeight: getInt64(p, prefix+"_DEFAULTVIEWPORTHEIGHT", 720),
		DefaultViewportWidth:  getInt64(p, prefix+"_DEFAULTVIEWPORTWIDTH", 1280),
		ViewportMargin:        getInt64(p, prefix+"_VIEWPORTMARGIN", 64),
		DisplayDensity:        getFloat64(p, prefix+"_DISPLAYDENSITY", 1.0),
		ResourcesDirectory:    p.Get(prefix+"_RESOURCESDIRECTORY", "resources"),
		ScreenshotDelay:       screenshotDelay,
	}, nil
}

// RequestLoggingConfig controls request logging.
type RequestLoggingConfig struct {
	Enable   bool `envconfig:"REQUESTLOGGING_ENABLE"`
	DumpBody bool `envconfig:"REQUESTLOGGING_DUMPBODY"`
}

func newRequestLoggingConfig(p Provider) *RequestLoggingConfig {
	const prefix = "REQUESTLOGGING"

	return &RequestLoggingConfig{
		Enable:   getBool(p, prefix+"_ENABLE", false),
		DumpBody: getBool(p, prefix+"_DUMPBODY", false),
	}
}

// TestAPIConfig controls test API behavior.
type TestAPIConfig struct {
	ClientKey string `envconfig:"TESTAPI_CLIENTKEY"`
	Enable    bool   `envconfig:"TESTAPI_ENABLE"`
}

func newTestAPIConfig(p Provider) (*TestAPIConfig, error) {
	const prefix = "TESTAPI"

	e := getBool(p, prefix+"_ENABLE", false)

	k := p.Get(prefix+"_CLIENTKEY", "")
	if e && k == "" {
		return nil, fmt.Errorf("either client key must be set or test API disabled")
	}

	return &TestAPIConfig{
		ClientKey: k,
		Enable:    e,
	}, nil
}

// AWSConfig keeps what's needed to communicate w/ AWS.
type AWSConfig struct {
	AccessKeyID string `envconfig:"AWS_ACCESSKEYID"`
	AccessKey   string `envconfig:"AWS_ACCESSKEY"`
	Region      string `envconfig:"AWS_REGION"`
	LogRequests bool   `envconfig:"AWS_LOGREQUESTS"`
}

func newAWSConfig(p Provider, m *MessageQueueConfig) (*AWSConfig, error) {
	const prefix = "AWS"

	i := p.Get(prefix+"_ACCESSKEYID", "")
	k := p.Get(prefix+"_ACCESSKEY", "")
	r := p.Get(prefix+"_REGION", "")
	if m.Implementation == MQSQS && (i == "" || k == "" || r == "") {
		return nil, fmt.Errorf("access key must be set")
	}

	return &AWSConfig{
		AccessKeyID: p.Get(prefix+"_ACCESSKEYID", ""),
		AccessKey:   p.Get(prefix+"_ACCESSKEY", ""),
		Region:      p.Get(prefix+"_REGION", ""),
		LogRequests: getBool(p, prefix+"_LOGREQUESTS", false),
	}, nil
}

// MessageQueueImplementation controls message queue implementation.
type MessageQueueImplementation string

const (
	// MQInProcess enables in-process implementation for testing purposes.
	MQInProcess MessageQueueImplementation = "inProcess"
	// MQSQS enables SQS implementation.
	MQSQS MessageQueueImplementation = "sqs"
)

func parseImplementation(s string) (MessageQueueImplementation, error) {
	switch MessageQueueImplementation(s) {
	case MQInProcess, MQSQS:
		return MessageQueueImplementation(s), nil

	default:
		return "", fmt.Errorf("unknown message queue implementation: %v", s)
	}
}

// MessageQueueConfig controls MQ behavior.
type MessageQueueConfig struct {
	Implementation  MessageQueueImplementation `envconfig:"MQ_IMPLEMENTATION"`
	URL             string                     `envconfig:"MQ_URL"`
	BatchSize       uint                       `envconfig:"MQ_BATCHSIZE"`
	PollingInterval time.Duration              `envconfig:"MQ_POLLINGINTERVAL"`
}

func newMessageQueueConfig(p Provider) (*MessageQueueConfig, error) {
	const prefix = "MQ"
	v := p.Get(prefix+"_IMPLEMENTATION", "")
	if v == "" {
		return nil, fmt.Errorf("message queue implementation must be set")
	}

	i, err := parseImplementation(v)
	if err != nil {
		return nil, err
	}

	return &MessageQueueConfig{
		Implementation:  i,
		URL:             p.Get(prefix+"_URL", ""),
		BatchSize:       getUint(p, prefix+"BATCHSIZE", 8),
		PollingInterval: getDuration(p, prefix+"POLLINGINTERVAL", 20*time.Second),
	}, nil
}

// MessageHandlerConfig controls message handler behavior.
type MessageHandlerConfig struct {
	ConcurrencyLevel uint `envconfig:"MESSAGEHANDLER_CONCURRENCYLEVEL"`
}

func newMessageHandlerConfig(p Provider) (*MessageHandlerConfig, error) {
	const prefix = "MESSAGEHANDLER"

	l := getUint(p, prefix+"_CONCURRENCYLEVEL", 8)
	if l == 0 {
		return nil, fmt.Errorf("concurrency level must be set")
	}

	return &MessageHandlerConfig{
		ConcurrencyLevel: l,
	}, nil
}

// Provider represents a configuration store backed by a key-value mapping.
type Provider interface {
	Get(key, fallback string) string
}

type dotenvProvider struct {
	values map[string]string
}

// NewDotenvProvider creates a .env file-backed Provider.
func NewDotenvProvider(filepath string) (Provider, error) {
	vs, err := godotenv.Read(filepath)
	if err != nil {
		return nil, err
	}

	return &dotenvProvider{
		values: vs,
	}, nil
}

func (p *dotenvProvider) Get(key, fallback string) string {
	v, ok := p.values[key]
	if ok {
		return v
	}

	return fallback
}

type providerChain struct {
	providers []Provider
}

// NewProviderChain allows value overriding by chaining multiple Provider.
func NewProviderChain(ps ...Provider) Provider {
	return &providerChain{
		providers: ps,
	}
}

func (c *providerChain) Get(key, fallback string) string {
	for _, p := range c.providers {
		v := p.Get(key, fallback)
		if v != fallback {
			return v
		}
	}

	return fallback
}

// NewBaseConfig creates a BaseConfig.
func NewBaseConfig(p Provider) (*BaseConfig, error) {
	c := BaseConfig{
		DB: &DatabaseConfig{
			Port:           p.Get("DB_PORT", "3306"),
			Host:           p.Get("DB_HOST", "localhost"),
			Name:           p.Get("DB_NAME", "spbibot"),
			Username:       p.Get("DB_USERNAME", "user"),
			UserPwd:        p.Get("DB_USERNAME_PWD", ""),
			Timeout:        getInt(p, "TIMEOUT", 0),
			UserIDHashCost: getInt(p, "USERID_HASH_COST", 10),
		},
		OAuthConfig: oauth.Config{
			Config: oauth2.Config{
				ClientID:     p.Get("CLIENT_ID", ""),
				ClientSecret: p.Get("CLIENT_SECRET", ""),
				RedirectURL:  p.Get("REDIRECTION_URL", ""),
				Endpoint:     getAzureADEndpoint(p),
			},
			LogoutEndpoint: getLogoutAzureADEndpoint(p),
			ResponseType:   p.Get("RESPONSE_TYPE", ""),
			ResponseMode:   p.Get("RESPONSE_MODE", ""),
			Resource:       p.Get("POWER_BI_RESOURCE", ""),
		},
		PowerBiClient: PowerBiClientConfig{
			APIURL:       p.Get("POWERBICLIENT_APIURL", "https://api.powerbi.com/v1.0/myorg"),
			ClientID:     p.Get("CLIENT_ID", ""),
			ClientSecret: p.Get("CLIENT_SECRET", ""),
			Endpoint:     getAzureADEndpoint(p),
		},
		BotAccessTokenConfig: oauth.Config{
			Config: oauth2.Config{
				ClientID:     p.Get("SLACK_CLIENT_ID", ""),
				ClientSecret: p.Get("SLACK_CLIENT_SECRET", ""),
				Endpoint: oauth2.Endpoint{
					AuthURL: slackOauthURL,
				},
				Scopes: getSlackAppScopes(),
			},
		},
		FeatureToggles: newFeatureTogglesConfig(p),
		AmplitudeKey:   p.Get("AMPLITUDE_API_KEY", ""),
	}

	h, err := newHostConfig(p)
	if err != nil {
		return nil, err
	}

	c.Host = h

	s, err := newSlackConfig(p)
	if err != nil {
		return nil, err
	}

	c.Slack = s

	l, err := newLoggerConfig(p)
	if err != nil {
		return nil, err
	}

	c.Logger = l

	b, err := newBrowserConfig(p)
	if err != nil {
		return nil, err
	}

	c.Browser = b

	m, err := newMessageQueueConfig(p)
	if err != nil {
		return nil, err
	}

	c.MessageQueue = m

	mh, err := newMessageHandlerConfig(p)
	if err != nil {
		return nil, err
	}

	c.MessageHandler = mh

	a, err := newAWSConfig(p, m)
	if err != nil {
		return nil, err
	}

	c.AWS = a

	return &c, nil
}

// NewBotConfig creates a BotConfig.
func NewBotConfig(p Provider) (*BotConfig, error) {
	base, err := NewBaseConfig(p)
	if err != nil {
		return nil, err
	}

	c := BotConfig{
		BaseConfig: base,
		Server: ServerConfig{
			Port:        getInt(p, "SERVER_PORT", 8080),
			TLSPort:     getInt(p, "SERVER_TLSPORT", 443),
			Host:        p.Get("SERVER_HOST", "localhost"),
			Certificate: p.Get("SERVER_CERTIFICATE", "certificate.pem"),
			Key:         p.Get("SERVER_KEY", "key.pem"),
		},
		RequestLogging: newRequestLoggingConfig(p),
	}

	t, err := newTestAPIConfig(p)
	if err != nil {
		return nil, err
	}

	c.TestAPI = t

	return &c, nil
}

func getSlackAppScopes() []string {
	return []string{"commands", "channels:read", "chat:write", "files:write", "users:read", "groups:read", "im:read", "users:read.email"}
}

func getAzureADEndpoint(p Provider) oauth2.Endpoint {
	url := msHost + p.Get("TENANT_ID", "common")

	return oauth2.Endpoint{
		AuthURL:  url + msAuthParam,
		TokenURL: url + msTokenParam,
	}
}

func getLogoutAzureADEndpoint(p Provider) oauth.LogoutEndpoint {
	url := msHost + p.Get("TENANT_ID", "common")

	return oauth.LogoutEndpoint{
		LogoutURL: url + msLogoutParam,
	}
}

func getInt(p Provider, key string, fallback int) int {
	v := p.Get(key, "")
	i, err := strconv.Atoi(v)
	if err == nil {
		return i
	}

	return fallback
}

func getBool(p Provider, key string, fallback bool) bool {
	v := p.Get(key, strconv.FormatBool(fallback))
	b, err := strconv.ParseBool(v)
	if err == nil {
		return b
	}

	return fallback
}

func getFloat64(p Provider, key string, fallback float64) float64 {
	v := p.Get(key, "")
	f64, err := strconv.ParseFloat(v, 64)
	if err == nil {
		return f64
	}

	return fallback
}

func getInt64(p Provider, key string, fallback int64) int64 {
	v := p.Get(key, "")
	i64, err := strconv.ParseInt(v, 10, 64)
	if err == nil {
		return i64
	}

	return fallback
}

func getUint(p Provider, key string, fallback uint) uint {
	v := p.Get(key, "")
	u, err := strconv.ParseUint(v, 10, 0)
	if err == nil {
		return uint(u)
	}

	return fallback
}

func getDuration(p Provider, key string, fallback time.Duration) time.Duration {
	v := p.Get(key, "")
	d, err := time.ParseDuration(v)
	if err == nil {
		return d
	}

	return fallback
}
