package tracker

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/grafana/pyroscope-go"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type ConfigTag string

const (
	PauseMetrics             ConfigTag = "pauseMetrics"             // Boolean - disable all metrics
	PauseDefaultMetrics      ConfigTag = "pauseDefaultMetrics"      // Boolean - disable default runtime metrics
	PauseTraces              ConfigTag = "pauseTraces"              // Boolean - disable all traces
	PauseLogs                ConfigTag = "pauseLogs"                // Boolean - disable all logs
	PauseProfiling           ConfigTag = "pauseProfiling"           // Boolean - disable profiling
	Debug                    ConfigTag = "debug"                    // Boolean - enable debug in console
	DebugLogFile             ConfigTag = "debugLogFile"             // Boolean - get logs files for debug mode
	Service                  ConfigTag = "service"                  // String - Service Name e.g: "My-Service"
	Target                   ConfigTag = "target"                   // String - Target e.g: "http://mxl-otelcol.service-availability.apps.ocpthikadev01.safaricom.net"
	Workspace                ConfigTag = "workspaceId"              // String - Workspace Name e.g: "My-Workspace"
	Token                    ConfigTag = "accessToken"              // String - Token string found at agent installation
	CustomResourceAttributes ConfigTag = "customResourceAttributes" // map[string]interface{}
)

type Config struct {
	serviceName              string
	workspaceId              string
	host                     string
	pauseMetrics             bool
	pauseDefaultMetrics      bool
	customResourceAttributes map[string]interface{}
	pauseTraces              bool
	pauseLogs                bool
	settings                 map[ConfigTag]interface{}
	pauseProfiling           bool
	debug                    bool
	debugLogFile             bool
	TenantID                 string
	AccessToken              string
	target                   string
	LogHost                  string
	fluentHost               string
	isServerless             string
	Tp                       *sdktrace.TracerProvider
	Mp                       *sdkmetric.MeterProvider
	Lp                       *sdklog.LoggerProvider
}

type Options func(*Config)

func WithConfigTag(k ConfigTag, v interface{}) Options {
	return func(c *Config) {
		if c.settings == nil {
			c.settings = make(map[ConfigTag]interface{})
		}
		c.settings[k] = v
	}
}
func doesNotContainHTTP(s string) bool {
	return !(strings.Contains(s, "http://") || strings.Contains(s, "https://"))
}

func newConfig(opts ...Options) *Config {
	c := new(Config)
	c.pauseMetrics = false
	c.pauseDefaultMetrics = false
	c.pauseTraces = false
	c.pauseProfiling = false
	c.fluentHost = "localhost"
	c.LogHost = "localhost"
	profilingServerUrl := os.Getenv("MXL_PROFILING_SERVER_URL")
	MXL_AGENT_SERVICE := os.Getenv("MXL_AGENT_SERVICE")
	authUrl := os.Getenv("MXL_AUTH_URL")
	if authUrl == "" {
		authUrl = "http://mxl-server.service-availability.apps.ocpthikadev01.safaricom.net/auth"
	}

	pid := strconv.Itoa(os.Getpid())
	for _, fn := range opts {
		fn(c)
	}

	if len(c.customResourceAttributes) == 0 {
		if v, ok := c.settings["customResourceAttributes"]; ok {
			if s, ok := v.(map[string]interface{}); ok {
				c.customResourceAttributes = s
			}
		}
	}

	if !c.pauseMetrics {
		if v, ok := c.settings["pauseMetrics"]; ok {
			if s, ok := v.(bool); ok {
				c.pauseMetrics = s
			}
		}
		// To set pauseMetrics via MXL_APM_COLLECT_METRICS environment variable
		if parsedValue, err := strconv.ParseBool(os.Getenv("MXL_APM_COLLECT_METRICS")); err == nil {
			c.pauseMetrics = !parsedValue
		}
	}
	if !c.pauseDefaultMetrics {
		if v, ok := c.settings["pauseDefaultMetrics"]; ok {
			if s, ok := v.(bool); ok {
				c.pauseDefaultMetrics = s
			}
		}
	}
	if !c.pauseTraces {
		if v, ok := c.settings["pauseTraces"]; ok {
			if s, ok := v.(bool); ok {
				c.pauseTraces = s
			}
			// To set pauseTraces via MXL_APM_COLLECT_TRACES environment variable
			if parsedValue, err := strconv.ParseBool(os.Getenv("MXL_APM_COLLECT_TRACES")); err == nil {
				c.pauseTraces = !parsedValue
			}
		}
	}
	if !c.pauseLogs {
		if v, ok := c.settings["pauseLogs"]; ok {
			if s, ok := v.(bool); ok {
				c.pauseLogs = s
			}
		}
		// To set pauseLogs via MXL_APM_COLLECT_LOGS environment variable
		if parsedValue, err := strconv.ParseBool(os.Getenv("MXL_APM_COLLECT_LOGS")); err == nil {
			c.pauseLogs = !parsedValue
		}
	}
	if !c.pauseProfiling {
		if v, ok := c.settings["pauseProfiling"]; ok {
			if s, ok := v.(bool); ok {
				c.pauseProfiling = s
			}
		}
		// To set pauseProfiling via MXL_APM_COLLECT_PROFILING environment variable
		if parsedValue, err := strconv.ParseBool(os.Getenv("MXL_APM_COLLECT_PROFILING")); err == nil {
			c.pauseProfiling = !parsedValue
		}
	}
	if !c.debug {
		if v, ok := c.settings["debug"]; ok {
			if s, ok := v.(bool); ok {
				c.debug = s
			}
		}
	}
	if !c.debugLogFile {
		if v, ok := c.settings["debugLogFile"]; ok {
			if s, ok := v.(bool); ok {
				c.debugLogFile = s
			}
		}
	}

	if c.serviceName == "" {
		if v, ok := c.settings["service"]; ok {
			if s, ok := v.(string); ok {
				c.serviceName = s
			}
		} else {
			c.serviceName = "Service-" + pid
		}
		// To set service name via ENV, MXL_SERVICE_NAME will have a lower priority compared to OTEL_SERVICE_NAME
		if envServiceName := os.Getenv("OTEL_SERVICE_NAME"); envServiceName != "" {
			c.serviceName = envServiceName
		} else if envServiceName := os.Getenv("MXL_SERVICE_NAME"); envServiceName != "" {
			c.serviceName = envServiceName
		}
	}

	if c.target == "" {
		if v, ok := c.settings["target"]; ok {
			if s, ok := v.(string); ok {
				os.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "false")
				c.target = s
				target := s
				if doesNotContainHTTP(target) {
					target = "https://" + target
				}
				c.fluentHost = strings.Replace(target, ":443", "", 1)
				c.fluentHost = strings.Replace(c.fluentHost, "https://", "", 1)
				c.isServerless = "1"
			}
		} else {
			os.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
			c.target = "localhost:9319"
			c.isServerless = "0"
			healthAPITarget := "http://localhost:13133/healthcheck"
			if MXL_AGENT_SERVICE != "" {
				healthAPITarget, _ = url.JoinPath("http://"+MXL_AGENT_SERVICE+":13133", "healthcheck")
			}
			req, err := http.NewRequest("GET", healthAPITarget, nil)
			if err != nil {
				log.Println("Error creating request:", err)
			}
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				log.Println("[MXL-Agent-debug] [WARNING] MXL Agent Health Check is failing ...\nThis could be due to incorrect value of MXL_AGENT_SERVICE\nIgnore the warning if you are using MXL Agent older than 1.7.7 (You can confirm by running `mxl-agent version`)")
			}
			defer func() {
				if resp != nil {
					resp.Body.Close()
				}
			}()
		}
	}

	c.host = getHostValue("MXL_AGENT_SERVICE", c.target)
	if MXL_AGENT_SERVICE != "" {
		c.LogHost = MXL_AGENT_SERVICE
	}

	if c.workspaceId == "" {
		if v, ok := c.settings["workspaceId"]; ok {
			if s, ok := v.(string); ok {
				c.workspaceId = s
			}
		} else {
			c.workspaceId = "Workspace-" + pid
		}
	}

	if c.AccessToken == "" {
		if v, ok := c.settings["accessToken"]; ok {
			if s, ok := v.(string); ok {
				c.AccessToken = s
			}
		}
		// To set the AccessToken via MXL_API_KEY environment variable
		if envAccessToken := os.Getenv("MXL_API_KEY"); envAccessToken != "" {
			c.AccessToken = envAccessToken
		}
	}

	if !c.pauseProfiling && c.AccessToken == "" {
		log.Println("Middleware accessToken is required for Profiling")
	}

	if !c.pauseProfiling && c.AccessToken != "" {
		req, err := http.NewRequest("POST", authUrl, nil)
		if err != nil {
			log.Println("Error creating request:", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Bearer "+c.AccessToken)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Println("Error making auth request")
			return c
		}
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Println("Error reading Middleware auth API response")
				return c
			}
			var data map[string]interface{}
			err = json.Unmarshal([]byte(string(body)), &data)
			if err != nil {
				log.Println("Error parsing Middleware JSON")
				return c
			}
			if data["success"] == true {
				project_uid, ok := data["data"].(map[string]interface{})["project_uid"].(string)
				if !ok {
					log.Println("Failed to retrieve TenantID from  api response")
					return c
				}
				if profilingServerUrl == "" {
					profilingServerUrl, _ = url.JoinPath("http://mxl-server.service-availability.apps.ocpthikadev01.safaricom.net", "profiling")
				}
				c.TenantID = project_uid
				profilingServiceName := strings.ReplaceAll(c.serviceName, " ", "-")
				_, err := pyroscope.Start(pyroscope.Config{
					ApplicationName: profilingServiceName,
					ServerAddress:   profilingServerUrl,
					TenantID:        c.TenantID,
					ProfileTypes: []pyroscope.ProfileType{
						pyroscope.ProfileCPU,
						pyroscope.ProfileInuseObjects,
						pyroscope.ProfileAllocObjects,
						pyroscope.ProfileInuseSpace,
						pyroscope.ProfileAllocSpace,
					},
				})
				if err != nil {
					log.Println("failed to enable continuous profiling: ", err)
				}
			}
		}
	}
	return c
}

func getHostValue(key, defaultValue string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return defaultValue
	}
	return value + ":9319"
}
