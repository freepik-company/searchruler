package webserver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"prosimcorp.com/SearchRuler/internal/pools"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

// states is a map of the states of the rules and their respective status used for
// the rules API endpoint
var (
	states = map[string]string{
		"PendingFiring":    "pending",
		"PendingResolving": "pending",
		"Firing":           "firing",
		"Normal":           "resolved",
	}
)

// RunWebserver starts a webserver that serves the rule pages
func RunWebserver(ctx context.Context, webserverAddr string, rulesPool *pools.RulesStore) error {
	logger := log.FromContext(ctx)

	logger.Info(fmt.Sprintf("Starting webserver in %s", webserverAddr))

	// Get the path of templates folder with the HTML files
	_, b, _, _ := runtime.Caller(0)
	basePath := filepath.Dir(b)

	// If the environment is production, use the executable path as basePath
	if os.Getenv("APP_ENV") == "production" {
		// For production environments
		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("error getting the path of the executable: %w", err)
		}
		basePath = filepath.Dir(execPath)
	}

	// Usar la ruta del ejecutable como basePath
	templatePath := filepath.Join(basePath, "static/templates")
	publicPath := filepath.Join(basePath, "static/public")

	// Create a new Fiber app with the HTML template engine
	engine := html.New(templatePath, ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// Define the routes
	// "/" redirets to "/rules"
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/rules")
	})
	app.Get("/rules", getRules(rulesPool))
	app.Get("/api/rules", getRulesJSON(rulesPool))
	app.Get("/rules/:key", getRule(rulesPool))
	app.Static("/static", publicPath)

	// Start the webserver
	if err := app.Listen(webserverAddr); err != nil {
		return err
	}

	return nil
}

// getRule returns a handler function that renders the rule detail page
func getRule(rulesPool *pools.RulesStore) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		key := c.Params("key")

		// Get the rule from the pool
		rule, exists := rulesPool.Get(key)
		if !exists {
			return c.Status(fiber.StatusNotFound).SendString("Rule not found")
		}

		// Parse the YAML fields
		actionRef, err := yaml.Marshal(rule.SearchRule.Spec.ActionRef)
		if err != nil {
			actionRef = []byte("Error serializing ActionRef")
		}
		condition, err := yaml.Marshal(rule.SearchRule.Spec.Condition)
		if err != nil {
			condition = []byte("Error serializing Condition")
		}

		// Render the rule detail page
		return c.Render("rule_detail", fiber.Map{
			"Key":       key,
			"Rule":      rule,
			"Condition": condition,
			"ActionRef": actionRef,
		})
	}
}

// getRules returns a handler function that renders the rules page
func getRules(rulesPool *pools.RulesStore) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		return c.Render("rules", fiber.Map{
			"Rules": rulesPool.Store,
		})
	}
}

// getRulesJSON returns a handler function that returns the rules in JSON format
func getRulesJSON(rulesPool *pools.RulesStore) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {

		alerts := []map[string]interface{}{}

		for key, value := range rulesPool.Store {
			alert := map[string]interface{}{
				"labels": map[string]string{
					"alertname": key,
					"namespace": value.SearchRule.Namespace,
				},
				"annotations": map[string]string{
					"description": value.SearchRule.Spec.Description,
					"summary":     value.SearchRule.Spec.Description,
				},
				"state": states[value.State],
				"activeAt": func() string {
					if value.FiringTime.IsZero() {
						return ""
					}
					return value.FiringTime.String()
				}(),
			}

			alerts = append(alerts, alert)
		}

		return c.JSON(map[string]interface{}{
			"alerts": alerts,
		})
	}
}
