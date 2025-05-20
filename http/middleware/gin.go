package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func AttachGin(r *gin.Engine, serviceName string) {
	r.Use(otelgin.Middleware(serviceName))
}
