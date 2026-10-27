package auth

import "github.com/labstack/echo/v5"

// Handlers defines the auth HTTP operations used by route mapping.
type Handlers interface {
	AdminLogin() echo.HandlerFunc
	Refresh() echo.HandlerFunc
	Logout() echo.HandlerFunc
	Check() echo.HandlerFunc
}
