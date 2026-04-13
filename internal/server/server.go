package server
import ("encoding/json";"log";"net/http";"github.com/stockyard-dev/stockyard-surveyor/internal/store";"github.com/stockyard-dev/stockyard/bus")
type Server struct{db *store.DB;mux *http.ServeMux;limits Limits;bus *bus.Bus}
func New(db *store.DB,limits Limits,b *bus.Bus)*Server{s:=&Server{db:db,mux:http.NewServeMux(),limits:limits,bus:b}
s.mux.HandleFunc("GET /api/responses",s.list)
s.mux.HandleFunc("POST /api/responses",s.create)
s.mux.HandleFunc("GET /api/responses/{id}",s.get)
s.mux.HandleFunc("PUT /api/responses/{id}",s.update)
s.mux.HandleFunc("DELETE /api/responses/{id}",s.del)
s.mux.HandleFunc("GET /api/stats",s.stats)
s.mux.HandleFunc("GET /api/health",s.health)
s.mux.HandleFunc("GET /ui",s.dashboard);s.mux.HandleFunc("GET /ui/",s.dashboard);s.mux.HandleFunc("GET /",s.root);
s.mux.HandleFunc("GET /api/tier",func(w http.ResponseWriter,r *http.Request){wj(w,200,map[string]any{"tier":s.limits.Tier,"upgrade_url":"https://stockyard.dev/surveyor/"})})
return s}
func(s *Server)ServeHTTP(w http.ResponseWriter,r *http.Request){s.mux.ServeHTTP(w,r)}
func wj(w http.ResponseWriter,c int,v any){w.Header().Set("Content-Type","application/json");w.WriteHeader(c);json.NewEncoder(w).Encode(v)}
func we(w http.ResponseWriter,c int,m string){wj(w,c,map[string]string{"error":m})}
func(s *Server)root(w http.ResponseWriter,r *http.Request){if r.URL.Path!="/"{http.NotFound(w,r);return};http.Redirect(w,r,"/ui",302)}
func(s *Server)list(w http.ResponseWriter,r *http.Request){
    q:=r.URL.Query().Get("q")
    filters:=map[string]string{}
    if v:=r.URL.Query().Get("status");v!=""{filters["status"]=v}
    if v:=r.URL.Query().Get("source");v!=""{filters["source"]=v}
    if q!=""||len(filters)>0{wj(w,200,map[string]any{"responses":oe(s.db.Search(q,filters))});return}
    wj(w,200,map[string]any{"responses":oe(s.db.List())})
}
func(s *Server)create(w http.ResponseWriter,r *http.Request){if s.limits.MaxItems>0{items:=s.db.List();if len(items)>=s.limits.MaxItems{we(w,402,"Free tier limit reached. Upgrade at https://stockyard.dev/surveyor/");return}};var e store.FormResponse;json.NewDecoder(r.Body).Decode(&e);if e.FormName==""{we(w,400,"name required");return};s.db.Create(&e);created:=s.db.Get(e.ID);s.publishSubmission(created);wj(w,201,created)}
func(s *Server)get(w http.ResponseWriter,r *http.Request){e:=s.db.Get(r.PathValue("id"));if e==nil{we(w,404,"not found");return};wj(w,200,e)}
func(s *Server)update(w http.ResponseWriter,r *http.Request){
    existing:=s.db.Get(r.PathValue("id"));if existing==nil{we(w,404,"not found");return}
    var patch store.FormResponse;json.NewDecoder(r.Body).Decode(&patch);patch.ID=existing.ID;patch.CreatedAt=existing.CreatedAt
    if patch.FormName==""{patch.FormName=existing.FormName}
    s.db.Update(&patch);wj(w,200,s.db.Get(patch.ID))
}
func(s *Server)del(w http.ResponseWriter,r *http.Request){s.db.Delete(r.PathValue("id"));wj(w,200,map[string]string{"deleted":"ok"})}
func(s *Server)stats(w http.ResponseWriter,r *http.Request){wj(w,200,s.db.Stats())}
func(s *Server)health(w http.ResponseWriter,r *http.Request){wj(w,200,map[string]any{"status":"ok","service":"surveyor","responses":s.db.Count()})}
func oe[T any](s []T)[]T{if s==nil{return[]T{}};return s}

// publishSubmission fires form.submitted on the bus. No-op when bus
// is nil (standalone mode). Runs in a goroutine so HTTP responses
// never block on bus writes. Errors are logged, never surfaced.
// Payload shape locked by docs/BUS-TOPICS.md v1 in stockyard-desktop.
func (s *Server) publishSubmission(e *store.FormResponse) {
	if s.bus == nil || e == nil {
		return
	}
	// Answers is stored as a string (raw JSON blob as submitted). Try
	// to pass through as structured data so subscribers don't have to
	// double-decode; fall back to the raw string on parse failure.
	var answers any = e.Answers
	var parsed map[string]any
	if err := json.Unmarshal([]byte(e.Answers), &parsed); err == nil {
		answers = parsed
	}
	payload := map[string]any{
		"submission_id": e.ID,
		"form_name":     e.FormName,
		"respondent":    e.Respondent,
		"answers":       answers,
		"source":        e.Source,
		"status":        e.Status,
		"submitted_at":  e.SubmittedAt,
	}
	go func() {
		if _, err := s.bus.Publish("form.submitted", payload); err != nil {
			log.Printf("surveyor: bus publish form.submitted failed: %v", err)
		}
	}()
}

func init(){log.SetFlags(log.LstdFlags|log.Lshortfile)}
