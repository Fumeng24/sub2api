package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
)

// AdminOrSupportAuthMiddleware allows super admins and support agents.
type AdminOrSupportAuthMiddleware gin.HandlerFunc

// CustomProviderSet contains site-specific middleware providers.
var CustomProviderSet = wire.NewSet(NewAdminOrSupportAuthMiddleware)
