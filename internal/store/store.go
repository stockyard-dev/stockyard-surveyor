package store
import ("database/sql";"fmt";"os";"path/filepath";"time";_ "modernc.org/sqlite")
type DB struct{db *sql.DB}
type FormResponse struct {
	ID string `json:"id"`
	FormName string `json:"name"`
	Respondent string `json:"respondent"`
	Answers string `json:"answers"`
	Score int `json:"score"`
	Status string `json:"status"`
	Source string `json:"source"`
	SubmittedAt string `json:"submitted_at"`
	CreatedAt string `json:"created_at"`
}
func Open(d string)(*DB,error){if err:=os.MkdirAll(d,0755);err!=nil{return nil,err};db,err:=sql.Open("sqlite",filepath.Join(d,"surveyor.db")+"?_journal_mode=WAL&_busy_timeout=5000");if err!=nil{return nil,err}
db.Exec(`CREATE TABLE IF NOT EXISTS responses(id TEXT PRIMARY KEY,name TEXT NOT NULL,respondent TEXT DEFAULT '',answers TEXT DEFAULT '{}',score INTEGER DEFAULT 0,status TEXT DEFAULT 'completed',source TEXT DEFAULT '',submitted_at TEXT DEFAULT '',created_at TEXT DEFAULT(datetime('now')))`)
return &DB{db:db},nil}
func(d *DB)Close()error{return d.db.Close()}
func genID()string{return fmt.Sprintf("%d",time.Now().UnixNano())}
func now()string{return time.Now().UTC().Format(time.RFC3339)}
func(d *DB)Create(e *FormResponse)error{e.ID=genID();e.CreatedAt=now();_,err:=d.db.Exec(`INSERT INTO responses(id,name,respondent,answers,score,status,source,submitted_at,created_at)VALUES(?,?,?,?,?,?,?,?,?)`,e.ID,e.FormName,e.Respondent,e.Answers,e.Score,e.Status,e.Source,e.SubmittedAt,e.CreatedAt);return err}
func(d *DB)Get(id string)*FormResponse{var e FormResponse;if d.db.QueryRow(`SELECT id,name,respondent,answers,score,status,source,submitted_at,created_at FROM responses WHERE id=?`,id).Scan(&e.ID,&e.FormName,&e.Respondent,&e.Answers,&e.Score,&e.Status,&e.Source,&e.SubmittedAt,&e.CreatedAt)!=nil{return nil};return &e}
func(d *DB)List()[]FormResponse{rows,_:=d.db.Query(`SELECT id,name,respondent,answers,score,status,source,submitted_at,created_at FROM responses ORDER BY created_at DESC`);if rows==nil{return nil};defer rows.Close();var o []FormResponse;for rows.Next(){var e FormResponse;rows.Scan(&e.ID,&e.FormName,&e.Respondent,&e.Answers,&e.Score,&e.Status,&e.Source,&e.SubmittedAt,&e.CreatedAt);o=append(o,e)};return o}
func(d *DB)Update(e *FormResponse)error{_,err:=d.db.Exec(`UPDATE responses SET name=?,respondent=?,answers=?,score=?,status=?,source=?,submitted_at=? WHERE id=?`,e.FormName,e.Respondent,e.Answers,e.Score,e.Status,e.Source,e.SubmittedAt,e.ID);return err}
func(d *DB)Delete(id string)error{_,err:=d.db.Exec(`DELETE FROM responses WHERE id=?`,id);return err}
func(d *DB)Count()int{var n int;d.db.QueryRow(`SELECT COUNT(*) FROM responses`).Scan(&n);return n}

func(d *DB)Search(q string, filters map[string]string)[]FormResponse{
    where:="1=1"
    args:=[]any{}
    if q!=""{
        where+=" AND (name LIKE ?)"
        args=append(args,"%"+q+"%");
    }
    if v,ok:=filters["status"];ok&&v!=""{where+=" AND status=?";args=append(args,v)}
    if v,ok:=filters["source"];ok&&v!=""{where+=" AND source=?";args=append(args,v)}
    rows,_:=d.db.Query(`SELECT id,name,respondent,answers,score,status,source,submitted_at,created_at FROM responses WHERE `+where+` ORDER BY created_at DESC`,args...)
    if rows==nil{return nil};defer rows.Close()
    var o []FormResponse;for rows.Next(){var e FormResponse;rows.Scan(&e.ID,&e.FormName,&e.Respondent,&e.Answers,&e.Score,&e.Status,&e.Source,&e.SubmittedAt,&e.CreatedAt);o=append(o,e)};return o
}

func(d *DB)Stats()map[string]any{
    m:=map[string]any{"total":d.Count()}
    rows,_:=d.db.Query(`SELECT status,COUNT(*) FROM responses GROUP BY status`)
    if rows!=nil{defer rows.Close();by:=map[string]int{};for rows.Next(){var s string;var c int;rows.Scan(&s,&c);by[s]=c};m["by_status"]=by}
    return m
}
