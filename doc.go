package forgerouter

import (
	"net/http"
)

func (r *ForgeRouter) addSwaggerUIEndpoint() {
	r.GET("/openapi/swagger", func(w http.ResponseWriter, req *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head>
    <title>API Documentation - Swagger UI</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.10.5/swagger-ui.css" />
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5.10.5/swagger-ui-bundle.js"></script>
    <script>
        SwaggerUIBundle({
            url: '/openapi.json',
            dom_id: '#swagger-ui',
            presets: [
                SwaggerUIBundle.presets.apis,
                SwaggerUIBundle.presets.standalone
            ],
            plugins: [
                SwaggerUIBundle.plugins.DownloadUrl
            ],
            deepLinking: true,
            showExtensions: true,
            showCommonExtensions: true,
            tryItOutEnabled: true,
            requestInterceptor: function(request) {
                // Add any request modifications here
                return request;
            },
            responseInterceptor: function(response) {
                // Add any response modifications here  
                return response;
            },
            supportedSubmitMethods: ['get', 'post', 'put', 'delete', 'patch', 'head', 'options'],
            // OpenAPI 3.1.1 specific configuration
            validatorUrl: null, // Disable validator for 3.1.1
            docExpansion: 'list', // 'list', 'full', 'none'
            defaultModelsExpandDepth: 1,
            defaultModelExpandDepth: 1,
            displayOperationId: false,
            displayRequestDuration: true,
            filter: true,
            showMutatedRequest: true,
            syntaxHighlight: {
                activated: true,
                theme: "agate"
            }
        });
    </script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	})
}
