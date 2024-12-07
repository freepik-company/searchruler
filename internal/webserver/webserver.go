package webserver

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"

	"prosimcorp.com/SearchRuler/internal/pools"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

var (
	states = map[string]string{
		"PendingFiring":    "pending",
		"PendingResolving": "pending",
		"Firing":           "firing",
		"Normal":           "resolved",
	}
)

func RunWebserver(ctx context.Context, config WebserverConfig, rulesPool *pools.RulesStore, alertsPool *pools.AlertsStore) error {
	logger := log.FromContext(ctx)

	logger.Info(fmt.Sprintf("Starting webserver in %s:%d", config.ListenAddr, config.Port))

	// Get the path of templates folder with the HTML files
	_, b, _, _ := runtime.Caller(0)
	basePath := filepath.Dir(b)
	templatePath := filepath.Join(basePath, "templates")

	// Create a new Fiber app with the HTML template engine
	engine := html.New(templatePath, ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// Define the routes
	app.Get("/rules", getRules(rulesPool))
	app.Get("/api/rules", getRulesJSON(rulesPool))
	app.Get("/rules/:key", getRule(rulesPool, alertsPool))

	// Start the webserver
	if err := app.Listen(fmt.Sprintf("%s:%d", config.ListenAddr, config.Port)); err != nil {
		return err
	}

	return nil
}

// getRule returns a handler function that renders the rule detail page
func getRule(rulesPool *pools.RulesStore, alertsPool *pools.AlertsStore) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		key := c.Params("key")

		// Get the rule from the pool
		rule, exists := rulesPool.Get(key)
		if !exists {
			return c.Status(fiber.StatusNotFound).SendString("Rule not found")
		}

		// Get the alert from the pool
		alert, exists := alertsPool.Get(key)
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
			"Key":            key,
			"State":          rule.State,
			"FiringTime":     rule.FiringTime,
			"ResolvingTime":  rule.ResolvingTime,
			"Description":    rule.SearchRule.Spec.Description,
			"QueryConnector": rule.SearchRule.Spec.QueryConnectorRef.Name,
			"CheckInterval":  rule.SearchRule.Spec.CheckInterval,
			"Query":          rule.SearchRule.Spec.Elasticsearch.Query.Raw,
			"Index":          rule.SearchRule.Spec.Elasticsearch.Index,
			"Condition":      condition,
			"ActionRef":      actionRef,
			"ConditionField": rule.SearchRule.Spec.Elasticsearch.ConditionField,
			"Value":          rule.Value,
			"Aggregations":   alert})
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
