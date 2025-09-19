package routes

import (
	"embed"
	"monolith/app/controllers"
	"monolith/app/middleware"
	"monolith/ws"
	"net/http"
)

func InitServerHandler(staticFiles embed.FS) http.Handler {
	// Create a new ServeMux for routing
	mux := http.NewServeMux()

	// Register all routes
	registerRoutes(mux, staticFiles)

	// apply all registered middleware
	middlewares := middleware.GetAllRegisteredMiddleware()
	var handler http.Handler
	for _, m := range middlewares {
		handler = m(mux)
	}

	return handler
}

func registerRoutes(mux *http.ServeMux, staticFiles embed.FS) {
	// Serve static files from embedded filesystem
	staticFileServer := http.FileServer(http.FS(staticFiles))
	mux.Handle("GET /static/", staticFileServer)

	mux.HandleFunc("GET /", controllers.IndexCtrl.ShowIndex)

	// serve websockets routes at "/ws" endpoint
	mux.HandleFunc("GET /ws", func(w http.ResponseWriter, r *http.Request) {
		ws.ServeWs(w, r)
	})

}
