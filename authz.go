package authz

import (
	"net/http"

	"log"

	"github.com/casbin/casbin"
	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyhttp/httpserver"
)

// Authorizer is a middleware for filtering clients based on their ip or country's ISO code.
type Authorizer struct {
	Next     httpserver.Handler
	Enforcer *casbin.Enforcer
}

// Init initializes the plugin
func init() {
	log.Printf("Vafan")
	caddy.RegisterPlugin("authz", caddy.Plugin{
		ServerType: "http",
		Action:     Setup,
	})
}

// GetConfig gets the config path that corresponds to c.
func GetConfig(c *caddy.Controller) (string, string) {
	modelPath := ""
	policyPath := ""
	for c.Next() { // skip the directive name
		if !c.NextArg() { // expect at least one value
			return c.ArgErr().Error(), policyPath // otherwise it's an error
		}
		modelPath = c.Val() // use the value

		if !c.NextArg() { // expect at least one value
			return modelPath, c.ArgErr().Error() // otherwise it's an error
		}
		policyPath = c.Val() // use the value
	}
	return modelPath, policyPath
}

// Setup parses the Casbin configuration and returns the middleware handler.
func Setup(c *caddy.Controller) error {
	modelPath, policyPath := GetConfig(c)
	e := casbin.NewEnforcer(modelPath, policyPath)

	// Create new middleware
	newMiddleWare := func(next httpserver.Handler) httpserver.Handler {
		return &Authorizer{
			Next:     next,
			Enforcer: e,
		}
	}
	// Add middleware
	cfg := httpserver.GetConfig(c)
	cfg.AddMiddleware(newMiddleWare)

	return nil
}

// ServeHTTP serves the request.
func (a Authorizer) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	if !a.CheckPermission(r) {
		w.WriteHeader(403)
		return http.StatusForbidden, nil
	} else {
		return a.Next.ServeHTTP(w, r)
	}
}

// GetUserName gets the user name from the request.
// Currently, only HTTP basic authentication is supported
func (a *Authorizer) GetUserName(r *http.Request) string {
	username := r.Header.Get("X-Forwarded-User")
	log.Printf("[INFO] X-Forwarded-User: %s", username)
	return username
}

// CheckPermission checks the user/method/path combination from the request.
// Returns true (permission granted) or false (permission forbidden)
func (a *Authorizer) CheckPermission(r *http.Request) bool {
	user := a.GetUserName(r)
	method := r.Method
	path := r.URL.Path
	return a.Enforcer.Enforce(user, path, method)
}
