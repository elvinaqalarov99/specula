package server

import (
	"net/http"
)

// swaggerUIHandler returns an http.Handler that serves a minimal Swagger UI
// pointing at the local /spec endpoint. No CDN — all inline.
func swaggerUIHandler() http.Handler {
	html := `<!DOCTYPE html>
<html>
<head>
  <title>API Docs — Live</title>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" >
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"> </script>
<script>
const ui = SwaggerUIBundle({
  url: "/spec",
  dom_id: '#swagger-ui',
  presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
  layout: "BaseLayout",
  deepLinking: true,
});

// Live reload via WebSocket
const wsProto = location.protocol === 'https:' ? 'wss' : 'ws';
const ws = new WebSocket(wsProto + '://' + location.host + '/ws');
ws.onmessage = (evt) => {
  const msg = JSON.parse(evt.data);
  if (msg.event === 'spec_update') {
    ui.specActions.updateSpec(JSON.stringify(msg.spec));
  }
};
ws.onclose = () => { setTimeout(() => location.reload(), 2000); };
</script>
</body>
</html>`

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	})
}
