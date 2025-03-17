package otel

const (
	SERVICE_NAME       = "deeppick"
	SERVICE_VERSION    = "v0.1.0"
	DEPLOY_ENVIRONMENT = "production"
	HTTP_ENDPOINT      = "tracing-analysis-dc-sh.aliyuncs.com"
	HTTP_URL_PATH      = "adapt_h6d9z5mhxp@36e4371eb6c0a0f_h6d9z5mhxp@53df7ad2afe8301/api/otlp/traces"
)

type Config struct {
	ServiceName       string
	ServiceVersion    string
	DeployEnvironment string
	HTTPEndpoint      string
	HTTPUrlPath       string
}

func DefaultConfig() *Config {
	return &Config{
		ServiceName:       SERVICE_NAME,
		ServiceVersion:    SERVICE_VERSION,
		DeployEnvironment: DEPLOY_ENVIRONMENT,
		HTTPEndpoint:      HTTP_ENDPOINT,
		HTTPUrlPath:       HTTP_URL_PATH,
	}
}

type Option func(c *Config)

func WithServiceName(serviceName string) Option {
	return func(c *Config) {
		c.ServiceName = serviceName
	}
}

func WithServiceVersion(version string) Option {
	return func(c *Config) {
		c.ServiceVersion = version
	}
}

func WithDeployEnvironment(env string) Option {
	return func(c *Config) {
		c.DeployEnvironment = env
	}
}

func WithHTTPEndpoint(endpoint string) Option {
	return func(c *Config) {
		c.HTTPEndpoint = endpoint
	}
}

func WithHTTPUrlPath(urlPath string) Option {
	return func(c *Config) {
		c.HTTPUrlPath = urlPath
	}
}
