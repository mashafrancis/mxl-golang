package mxlgolang

import (
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
)

type Config struct {
	oTLPEndpoint        string
	insecure            bool
	disableTraces       bool
	disableMetrics      bool
	disableLogs         bool
	authorizationHeader string
	serviceName         string
	attributes          []attribute.KeyValue
	headers             map[string]string
	useHTTP             bool
}

func NewConfig() *Config {
	return &Config{
		oTLPEndpoint: "localhost:4317",
		insecure:     true,
		serviceName:  "example-service",
		attributes: []attribute.KeyValue{
			attribute.String("service.name", "example-service"),
			attribute.String("library.language", "go"),
		},
		headers: make(map[string]string),
	}
}

func (c *Config) SetHeader(header string, value string) *Config {
	c.headers[header] = value
	return c
}

func (c *Config) SetUseHttpOverGrpc(useHTTP bool) *Config {
	c.useHTTP = useHTTP
	return c
}

func (c *Config) SetOTLPEndpoint(endpoint string) *Config {
	c.oTLPEndpoint = endpoint
	return c
}
func (c *Config) SetInsecure(insecure bool) *Config {
	c.insecure = insecure
	return c
}

func (c *Config) SetDisableTraces(disable bool) *Config {
	c.disableTraces = disable
	return c
}
func (c *Config) SetDisableMetrics(disable bool) *Config {
	c.disableMetrics = disable
	return c
}
func (c *Config) SetDisableLogs(disable bool) *Config {
	c.disableLogs = disable
	return c
}
func (c *Config) SetAuthorizationHeader(header string) *Config {
	c.authorizationHeader = header
	return c
}

func (c *Config) SetServiceName(name string) *Config {
	c.attributes[0] = attribute.String("service.name", name)
	return c
}

func (c *Config) SetResourceAttr(key string, value any) *Config {
	switch v := value.(type) {
	case string:
		c.attributes = append(c.attributes, attribute.String(key, v))
	case int:
		c.attributes = append(c.attributes, attribute.Int(key, v))
	case float64:
		c.attributes = append(c.attributes, attribute.Float64(key, v))

	default:
		slog.Warn("resource attributes should be of type string or int")

	}
	return c
}
