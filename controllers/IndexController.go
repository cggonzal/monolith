package controllers

import (
	"monolith/config"
	"monolith/views"
	"net/http"
)

type IndexController struct{}

var IndexCtrl = &IndexController{}

// ShowIndex renders the index page
func (ic *IndexController) ShowIndex(w http.ResponseWriter, r *http.Request) {
	// Only respond to the exact root path. Anything else should be a 404.
	if r.URL.Path != "/" {
		ErrorCtrl.Show404(w, r)
		return
	}

	data := map[string]interface{}{
		"monolith_version": config.MONOLITH_VERSION,
	}
	views.Render(w, "index.html.tmpl", data)
}
