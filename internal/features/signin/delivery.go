package signin

import "github.com/labstack/echo/v5"

// Handlers defines the Google sign-in HTTP operations used by route mapping.
type Handlers interface {
	GoogleSignIn() echo.HandlerFunc
}
