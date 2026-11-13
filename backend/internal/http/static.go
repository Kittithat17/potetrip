package http

import (
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

// contentSecurityPolicy limits the web app to its own code (fonts are
// self-hosted). The one exception is Google profile photos.
const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"font-src 'self'; " +
	"img-src 'self' data: blob: https://*.googleusercontent.com; " +
	"connect-src 'self'; " +
	"worker-src 'self'; manifest-src 'self'; " +
	"object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"

// spa serves the built web app from dir (production: frontend and API on one
// origin, as the session cookie requires). Real files are served as they
// are; any other path gets index.html so client routes like /t/ABC load the
// app. Unknown /api paths stay JSON 404s.
func spa(dir string, secure bool) http.Handler {
	root := os.DirFS(dir)
	files := http.FileServerFS(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeError(w, http.StatusNotFound, "not_found", "ไม่พบ API นี้")
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		h := w.Header()
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), payment=()")
		if secure {
			h.Set("Strict-Transport-Security", "max-age=31536000")
		}

		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		st, err := fs.Stat(root, name)
		if name != "" && err != nil && looksLikeFile(name) {
			// /.env, /.git/config, /x.js.map ...: say "not found" rather than
			// serving the app, so nobody mistakes it for a real file.
			http.NotFound(w, r)
			return
		}
		if name == "" || err != nil || st.IsDir() {
			// A client-side route: always the latest app shell.
			h.Set("Cache-Control", "no-cache")
			r = r.Clone(r.Context())
			r.URL.Path = "/"
			files.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(name, "assets/") {
			// Vite puts a content hash in these names, so they never change.
			h.Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			h.Set("Cache-Control", "no-cache") // sw.js, manifest, icons
		}
		files.ServeHTTP(w, r)
	})
}

// looksLikeFile: a dot-file or a name with an extension. App routes
// (/t/ABCDEFGH23/settle) have neither.
func looksLikeFile(name string) bool {
	for _, seg := range strings.Split(name, "/") {
		if strings.HasPrefix(seg, ".") {
			return true
		}
	}
	return path.Ext(name) != ""
}
