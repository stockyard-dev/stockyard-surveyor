package server

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/stockyard-dev/stockyard-surveyor/internal/store"
)

type Server struct {
	db  *store.DB
	mux *http.ServeMux
}

func New(db *store.DB) *Server {
	s := &Server{db: db, mux: http.NewServeMux()}

	// API
	s.mux.HandleFunc("GET /api/forms", s.listForms)
	s.mux.HandleFunc("POST /api/forms", s.createForm)
	s.mux.HandleFunc("GET /api/forms/{id}", s.getForm)
	s.mux.HandleFunc("PUT /api/forms/{id}", s.updateForm)
	s.mux.HandleFunc("DELETE /api/forms/{id}", s.deleteForm)
	s.mux.HandleFunc("GET /api/forms/{id}/responses", s.listResponses)
	s.mux.HandleFunc("GET /api/forms/{id}/responses/export", s.exportCSV)
	s.mux.HandleFunc("DELETE /api/responses/{id}", s.deleteResponse)
	s.mux.HandleFunc("GET /api/health", s.health)
	s.mux.HandleFunc("GET /api/stats", s.stats)

	// Public submission (both JSON API and form POST)
	s.mux.HandleFunc("POST /api/forms/{id}/responses", s.submitResponse)
	s.mux.HandleFunc("POST /f/{slug}", s.submitPublic)

	// Public form page
	s.mux.HandleFunc("GET /f/{slug}", s.publicForm)

	// Dashboard
	s.mux.HandleFunc("GET /ui", s.dashboard)
	s.mux.HandleFunc("GET /ui/", s.dashboard)
	s.mux.HandleFunc("GET /", s.root)

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.Redirect(w, r, "/ui", http.StatusFound)
		return
	}
	http.NotFound(w, r)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"status":  "ok",
		"service": "stockyard-surveyor",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"forms":     s.db.FormCount(),
		"responses": s.db.TotalResponseCount(),
	})
}

// ── Forms CRUD ──

func (s *Server) listForms(w http.ResponseWriter, r *http.Request) {
	forms := s.db.ListForms()
	if forms == nil {
		forms = []store.Form{}
	}
	writeJSON(w, map[string]any{"forms": forms, "count": len(forms)})
}

func (s *Server) createForm(w http.ResponseWriter, r *http.Request) {
	var f store.Form
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		writeErr(w, 400, "invalid JSON")
		return
	}
	if f.Title == "" {
		writeErr(w, 400, "title required")
		return
	}
	if f.Fields == nil {
		f.Fields = []store.Field{}
	}
	f.Active = true
	if err := s.db.CreateForm(&f); err != nil {
		writeErr(w, 500, fmt.Sprintf("create: %v", err))
		return
	}
	log.Printf("surveyor: created form %q (%s)", f.Title, f.ID)
	w.WriteHeader(201)
	writeJSON(w, f)
}

func (s *Server) getForm(w http.ResponseWriter, r *http.Request) {
	f := s.db.GetForm(r.PathValue("id"))
	if f == nil {
		writeErr(w, 404, "form not found")
		return
	}
	writeJSON(w, f)
}

func (s *Server) updateForm(w http.ResponseWriter, r *http.Request) {
	existing := s.db.GetForm(r.PathValue("id"))
	if existing == nil {
		writeErr(w, 404, "form not found")
		return
	}
	var update store.Form
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeErr(w, 400, "invalid JSON")
		return
	}
	if update.Title != "" {
		existing.Title = update.Title
	}
	if update.Description != "" {
		existing.Description = update.Description
	}
	if update.Slug != "" {
		existing.Slug = update.Slug
	}
	if update.Fields != nil {
		existing.Fields = update.Fields
	}
	if update.WebhookURL != "" {
		existing.WebhookURL = update.WebhookURL
	}
	existing.Active = update.Active

	if err := s.db.UpdateForm(existing); err != nil {
		writeErr(w, 500, fmt.Sprintf("update: %v", err))
		return
	}
	writeJSON(w, existing)
}

func (s *Server) deleteForm(w http.ResponseWriter, r *http.Request) {
	if err := s.db.DeleteForm(r.PathValue("id")); err != nil {
		writeErr(w, 500, fmt.Sprintf("delete: %v", err))
		return
	}
	writeJSON(w, map[string]string{"status": "deleted"})
}

// ── Responses ──

func (s *Server) listResponses(w http.ResponseWriter, r *http.Request) {
	formID := r.PathValue("id")
	f := s.db.GetForm(formID)
	if f == nil {
		writeErr(w, 404, "form not found")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 100
	}
	responses := s.db.ListResponses(formID, limit, offset)
	if responses == nil {
		responses = []store.Response{}
	}
	writeJSON(w, map[string]any{
		"responses": responses,
		"count":     len(responses),
		"total":     s.db.ResponseCount(formID),
	})
}

func (s *Server) submitResponse(w http.ResponseWriter, r *http.Request) {
	formID := r.PathValue("id")
	f := s.db.GetForm(formID)
	if f == nil {
		writeErr(w, 404, "form not found")
		return
	}
	if !f.Active {
		writeErr(w, 403, "form is not accepting responses")
		return
	}

	var resp store.Response
	resp.FormID = f.ID
	resp.IP = r.Header.Get("X-Forwarded-For")
	if resp.IP == "" {
		resp.IP = r.RemoteAddr
	}
	resp.UserAgent = r.Header.Get("User-Agent")

	if err := json.NewDecoder(r.Body).Decode(&resp.Data); err != nil {
		writeErr(w, 400, "invalid JSON — expected {\"field_name\": \"value\", ...}")
		return
	}

	if err := s.db.CreateResponse(&resp); err != nil {
		writeErr(w, 500, fmt.Sprintf("save: %v", err))
		return
	}

	// Fire webhook if configured
	if f.WebhookURL != "" {
		go fireWebhook(f.WebhookURL, f, &resp)
	}

	log.Printf("surveyor: response on form %q from %s", f.Title, resp.IP)
	w.WriteHeader(201)
	writeJSON(w, map[string]string{"status": "submitted", "id": resp.ID})
}

func (s *Server) submitPublic(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	f := s.db.GetForm(slug)
	if f == nil {
		http.Error(w, "Form not found", 404)
		return
	}
	if !f.Active {
		http.Error(w, "Form closed", 403)
		return
	}

	r.ParseForm()
	data := make(map[string]string)
	for key, vals := range r.PostForm {
		if key == "_redirect" {
			continue
		}
		data[key] = strings.Join(vals, ", ")
	}

	resp := store.Response{FormID: f.ID, Data: data, IP: r.RemoteAddr, UserAgent: r.Header.Get("User-Agent")}
	s.db.CreateResponse(&resp)

	if f.WebhookURL != "" {
		go fireWebhook(f.WebhookURL, f, &resp)
	}

	redirect := r.FormValue("_redirect")
	if redirect != "" {
		http.Redirect(w, r, redirect, http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!DOCTYPE html><html><head><title>Thank you</title><style>body{font-family:sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;background:#f5f5f5}div{text-align:center;padding:2rem}</style></head><body><div><h2>Thank you</h2><p>Your response has been recorded.</p></div></body></html>`)
}

func (s *Server) deleteResponse(w http.ResponseWriter, r *http.Request) {
	if err := s.db.DeleteResponse(r.PathValue("id")); err != nil {
		writeErr(w, 500, fmt.Sprintf("delete: %v", err))
		return
	}
	writeJSON(w, map[string]string{"status": "deleted"})
}

func (s *Server) exportCSV(w http.ResponseWriter, r *http.Request) {
	formID := r.PathValue("id")
	f := s.db.GetForm(formID)
	if f == nil {
		writeErr(w, 404, "form not found")
		return
	}

	responses := s.db.ListResponses(formID, 10000, 0)

	// Collect all field names
	fieldNames := make([]string, 0)
	seen := make(map[string]bool)
	for _, field := range f.Fields {
		if !seen[field.Name] {
			fieldNames = append(fieldNames, field.Name)
			seen[field.Name] = true
		}
	}
	// Also include any data keys not in field definitions
	for _, resp := range responses {
		for k := range resp.Data {
			if !seen[k] {
				fieldNames = append(fieldNames, k)
				seen[k] = true
			}
		}
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	header := append([]string{"id", "submitted_at"}, fieldNames...)
	writer.Write(header)

	for _, resp := range responses {
		row := []string{resp.ID, resp.CreatedAt.Format(time.RFC3339)}
		for _, name := range fieldNames {
			row = append(row, resp.Data[name])
		}
		writer.Write(row)
	}
	writer.Flush()

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-responses.csv", f.Slug))
	w.Write(buf.Bytes())
}

// ── Public form page ──

func (s *Server) publicForm(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	f := s.db.GetForm(slug)
	if f == nil {
		http.Error(w, "Form not found", 404)
		return
	}

	var fields string
	for _, field := range f.Fields {
		req := ""
		if field.Required {
			req = " required"
		}
		switch field.Type {
		case "textarea":
			fields += fmt.Sprintf(`<div class="field"><label>%s</label><textarea name="%s"%s rows="4"></textarea></div>`, field.Label, field.Name, req)
		case "select":
			opts := ""
			for _, o := range field.Options {
				opts += fmt.Sprintf(`<option value="%s">%s</option>`, o, o)
			}
			fields += fmt.Sprintf(`<div class="field"><label>%s</label><select name="%s"%s><option value="">Select...</option>%s</select></div>`, field.Label, field.Name, req, opts)
		case "checkbox":
			fields += fmt.Sprintf(`<div class="field"><label><input type="checkbox" name="%s" value="yes"%s> %s</label></div>`, field.Name, req, field.Label)
		case "radio":
			radios := ""
			for _, o := range field.Options {
				radios += fmt.Sprintf(`<label class="radio"><input type="radio" name="%s" value="%s"%s> %s</label>`, field.Name, o, req, o)
			}
			fields += fmt.Sprintf(`<div class="field"><label>%s</label><div>%s</div></div>`, field.Label, radios)
		default:
			inputType := field.Type
			if inputType == "" {
				inputType = "text"
			}
			fields += fmt.Sprintf(`<div class="field"><label>%s</label><input type="%s" name="%s"%s></div>`, field.Label, inputType, field.Name, req)
		}
	}

	desc := ""
	if f.Description != "" {
		desc = fmt.Sprintf(`<p class="desc">%s</p>`, f.Description)
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!DOCTYPE html><html><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s</title>
<style>*{margin:0;padding:0;box-sizing:border-box}body{font-family:-apple-system,sans-serif;background:#f5f5f5;min-height:100vh;display:flex;align-items:center;justify-content:center;padding:2rem}
.form-card{background:#fff;border-radius:8px;padding:2rem;max-width:500px;width:100%%;box-shadow:0 1px 3px rgba(0,0,0,.1)}
h1{font-size:1.4rem;margin-bottom:.5rem}.desc{color:#666;font-size:.9rem;margin-bottom:1.5rem}
.field{margin-bottom:1rem}.field label{display:block;font-size:.85rem;font-weight:500;margin-bottom:.3rem;color:#333}
.field input[type=text],.field input[type=email],.field input[type=number],.field textarea,.field select{width:100%%;padding:.5rem;border:1px solid #ddd;border-radius:4px;font-size:.9rem}
.field textarea{resize:vertical}.radio{display:block;margin:.3rem 0;font-weight:400}
button{background:#c45d2c;color:#fff;border:none;padding:.6rem 1.5rem;border-radius:4px;font-size:.9rem;cursor:pointer;margin-top:.5rem}button:hover{background:#e8753a}
.powered{text-align:center;margin-top:1.5rem;font-size:.7rem;color:#999}
</style></head><body><div class="form-card"><h1>%s</h1>%s<form method="POST" action="/f/%s">%s<button type="submit">Submit</button></form><div class="powered">Powered by <a href="https://stockyard.dev/surveyor/" style="color:#999">Surveyor</a></div></div></body></html>`,
		f.Title, f.Title, desc, f.Slug, fields)
}

// ── Webhook ──

func fireWebhook(url string, form *store.Form, resp *store.Response) {
	payload, _ := json.Marshal(map[string]any{
		"event":     "response.created",
		"form_id":   form.ID,
		"form_slug": form.Slug,
		"form_name": form.Title,
		"response":  resp,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		log.Printf("surveyor: webhook error: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Stockyard-Surveyor/1.0")
	client := &http.Client{Timeout: 10 * time.Second}
	resp2, err := client.Do(req)
	if err != nil {
		log.Printf("surveyor: webhook failed: %v", err)
		return
	}
	resp2.Body.Close()
	log.Printf("surveyor: webhook sent to %s (%d)", url, resp2.StatusCode)
}
