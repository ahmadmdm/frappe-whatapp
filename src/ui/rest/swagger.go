package rest

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
)

func RegisterSwaggerRoutes(app fiber.Router) {
	app.Get("/swagger", swaggerUI)
	app.Get("/swagger/openapi.yaml", swaggerSpec)
}

func swaggerUI(c *fiber.Ctx) error {
	c.Type("html", "utf-8")
	return c.SendString(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>WhatsApp API Swagger</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
  <style>
    html, body {
      margin: 0;
      background: #f5f7fb;
    }
    .topbar {
      display: none;
    }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    const basePath = window.location.pathname.replace(/\/+$/, '');
    SwaggerUIBundle({
      url: basePath + '/openapi.yaml',
      dom_id: '#swagger-ui',
      deepLinking: true,
      displayRequestDuration: true,
      persistAuthorization: true,
      tryItOutEnabled: true
    });
  </script>
</body>
</html>`)
}

func swaggerSpec(c *fiber.Ctx) error {
	content, path, err := loadSwaggerSpec()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  fiber.StatusInternalServerError,
			"code":    "SWAGGER_SPEC_NOT_FOUND",
			"message": "Unable to locate docs/openapi.yaml from the running workspace",
			"results": fiber.Map{"error": err.Error()},
		})
	}

	c.Set("X-Swagger-Source", path)
	c.Type("yaml", "utf-8")
	return c.Send(content)
}

func loadSwaggerSpec() ([]byte, string, error) {
	candidates := []string{
		filepath.Join("docs", "openapi.yaml"),
		filepath.Join("..", "docs", "openapi.yaml"),
		filepath.Join("..", "..", "docs", "openapi.yaml"),
	}

	for _, candidate := range candidates {
		content, err := os.ReadFile(candidate)
		if err == nil {
			absPath, absErr := filepath.Abs(candidate)
			if absErr != nil {
				absPath = candidate
			}
			return content, absPath, nil
		}
	}

	return nil, "", fmt.Errorf("checked %d candidate locations", len(candidates))
}
