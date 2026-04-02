package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct{ db *sql.DB }

type Form struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Fields      []Field   `json:"fields"`
	WebhookURL  string    `json:"webhook_url,omitempty"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	Responses   int       `json:"response_count"`
}

type Field struct {
	Name     string   `json:"name"`
	Label    string   `json:"label"`
	Type     string   `json:"type"` // text, textarea, email, number, select, checkbox, radio
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"` // for select, radio
}

type Response struct {
	ID        string            `json:"id"`
	FormID    string            `json:"form_id"`
	Data      map[string]string `json:"data"`
	IP        string            `json:"ip,omitempty"`
	UserAgent string            `json:"user_agent,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}
	dsn := filepath.Join(dataDir, "surveyor.db") + "?_journal_mode=WAL&_busy_timeout=5000"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	for _, q := range []string{
		`CREATE TABLE IF NOT EXISTS forms (
			id TEXT PRIMARY KEY,
			slug TEXT UNIQUE NOT NULL,
			title TEXT NOT NULL,
			description TEXT DEFAULT '',
			fields_json TEXT DEFAULT '[]',
			webhook_url TEXT DEFAULT '',
			active INTEGER DEFAULT 1,
			created_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS responses (
			id TEXT PRIMARY KEY,
			form_id TEXT NOT NULL REFERENCES forms(id),
			data_json TEXT NOT NULL,
			ip TEXT DEFAULT '',
			user_agent TEXT DEFAULT '',
			created_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_responses_form ON responses(form_id)`,
		`CREATE INDEX IF NOT EXISTS idx_forms_slug ON forms(slug)`,
	} {
		if _, err := db.Exec(q); err != nil {
			return nil, fmt.Errorf("migrate: %w", err)
		}
	}
	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }

func genID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// ── Forms ──

func (d *DB) CreateForm(f *Form) error {
	f.ID = genID()
	f.CreatedAt = time.Now().UTC()
	if f.Slug == "" {
		f.Slug = f.ID
	}
	fieldsJSON, _ := json.Marshal(f.Fields)
	_, err := d.db.Exec(`INSERT INTO forms (id, slug, title, description, fields_json, webhook_url, active) VALUES (?,?,?,?,?,?,?)`,
		f.ID, f.Slug, f.Title, f.Description, string(fieldsJSON), f.WebhookURL, boolInt(f.Active))
	return err
}

func (d *DB) UpdateForm(f *Form) error {
	fieldsJSON, _ := json.Marshal(f.Fields)
	_, err := d.db.Exec(`UPDATE forms SET slug=?, title=?, description=?, fields_json=?, webhook_url=?, active=? WHERE id=?`,
		f.Slug, f.Title, f.Description, string(fieldsJSON), f.WebhookURL, boolInt(f.Active), f.ID)
	return err
}

func (d *DB) DeleteForm(id string) error {
	_, err := d.db.Exec(`DELETE FROM responses WHERE form_id=?`, id)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(`DELETE FROM forms WHERE id=?`, id)
	return err
}

func (d *DB) GetForm(id string) *Form {
	row := d.db.QueryRow(`SELECT f.id, f.slug, f.title, f.description, f.fields_json, f.webhook_url, f.active, f.created_at,
		(SELECT COUNT(*) FROM responses r WHERE r.form_id = f.id)
		FROM forms f WHERE f.id=? OR f.slug=?`, id, id)
	return scanForm(row)
}

func (d *DB) ListForms() []Form {
	rows, err := d.db.Query(`SELECT f.id, f.slug, f.title, f.description, f.fields_json, f.webhook_url, f.active, f.created_at,
		(SELECT COUNT(*) FROM responses r WHERE r.form_id = f.id)
		FROM forms f ORDER BY f.created_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []Form
	for rows.Next() {
		f := scanFormRow(rows)
		if f != nil {
			result = append(result, *f)
		}
	}
	return result
}

func (d *DB) FormCount() int {
	var n int
	d.db.QueryRow(`SELECT COUNT(*) FROM forms`).Scan(&n)
	return n
}

// ── Responses ──

func (d *DB) CreateResponse(r *Response) error {
	r.ID = genID()
	r.CreatedAt = time.Now().UTC()
	dataJSON, _ := json.Marshal(r.Data)
	_, err := d.db.Exec(`INSERT INTO responses (id, form_id, data_json, ip, user_agent) VALUES (?,?,?,?,?)`,
		r.ID, r.FormID, string(dataJSON), r.IP, r.UserAgent)
	return err
}

func (d *DB) ListResponses(formID string, limit, offset int) []Response {
	if limit <= 0 {
		limit = 100
	}
	rows, err := d.db.Query(`SELECT id, form_id, data_json, ip, user_agent, created_at FROM responses WHERE form_id=? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		formID, limit, offset)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []Response
	for rows.Next() {
		var r Response
		var dataJSON, createdAt string
		if err := rows.Scan(&r.ID, &r.FormID, &dataJSON, &r.IP, &r.UserAgent, &createdAt); err != nil {
			continue
		}
		json.Unmarshal([]byte(dataJSON), &r.Data)
		r.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		result = append(result, r)
	}
	return result
}

func (d *DB) ResponseCount(formID string) int {
	var n int
	d.db.QueryRow(`SELECT COUNT(*) FROM responses WHERE form_id=?`, formID).Scan(&n)
	return n
}

func (d *DB) TotalResponseCount() int {
	var n int
	d.db.QueryRow(`SELECT COUNT(*) FROM responses`).Scan(&n)
	return n
}

func (d *DB) DeleteResponse(id string) error {
	_, err := d.db.Exec(`DELETE FROM responses WHERE id=?`, id)
	return err
}

// ── Helpers ──

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

type scannable interface {
	Scan(dest ...any) error
}

func scanForm(row scannable) *Form {
	var f Form
	var fieldsJSON, createdAt string
	var active int
	if err := row.Scan(&f.ID, &f.Slug, &f.Title, &f.Description, &fieldsJSON, &f.WebhookURL, &active, &createdAt, &f.Responses); err != nil {
		return nil
	}
	json.Unmarshal([]byte(fieldsJSON), &f.Fields)
	f.Active = active == 1
	f.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return &f
}

func scanFormRow(rows *sql.Rows) *Form {
	var f Form
	var fieldsJSON, createdAt string
	var active int
	if err := rows.Scan(&f.ID, &f.Slug, &f.Title, &f.Description, &fieldsJSON, &f.WebhookURL, &active, &createdAt, &f.Responses); err != nil {
		return nil
	}
	json.Unmarshal([]byte(fieldsJSON), &f.Fields)
	f.Active = active == 1
	f.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return &f
}
