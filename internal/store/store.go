package store
import ("database/sql";"fmt";"os";"path/filepath";"time";_ "modernc.org/sqlite")
type DB struct{db *sql.DB}
type Form struct{
	ID string `json:"id"`
	Title string `json:"title"`
	Description string `json:"description"`
	Fields string `json:"fields"`
	ResponseCount int `json:"response_count"`
	Status string `json:"status"`
	CreatedAt string `json:"created_at"`
}
func Open(d string)(*DB,error){if err:=os.MkdirAll(d,0755);err!=nil{return nil,err};db,err:=sql.Open("sqlite",filepath.Join(d,"surveyor.db")+"?_journal_mode=WAL&_busy_timeout=5000");if err!=nil{return nil,err}
db.Exec(`CREATE TABLE IF NOT EXISTS forms(id TEXT PRIMARY KEY,title TEXT NOT NULL,description TEXT DEFAULT '',fields TEXT DEFAULT '[]',response_count INTEGER DEFAULT 0,status TEXT DEFAULT 'active',created_at TEXT DEFAULT(datetime('now')))`)
return &DB{db:db},nil}
func(d *DB)Close()error{return d.db.Close()}
func genID()string{return fmt.Sprintf("%d",time.Now().UnixNano())}
func now()string{return time.Now().UTC().Format(time.RFC3339)}
func(d *DB)Create(e *Form)error{e.ID=genID();e.CreatedAt=now();_,err:=d.db.Exec(`INSERT INTO forms(id,title,description,fields,response_count,status,created_at)VALUES(?,?,?,?,?,?,?)`,e.ID,e.Title,e.Description,e.Fields,e.ResponseCount,e.Status,e.CreatedAt);return err}
func(d *DB)Get(id string)*Form{var e Form;if d.db.QueryRow(`SELECT id,title,description,fields,response_count,status,created_at FROM forms WHERE id=?`,id).Scan(&e.ID,&e.Title,&e.Description,&e.Fields,&e.ResponseCount,&e.Status,&e.CreatedAt)!=nil{return nil};return &e}
func(d *DB)List()[]Form{rows,_:=d.db.Query(`SELECT id,title,description,fields,response_count,status,created_at FROM forms ORDER BY created_at DESC`);if rows==nil{return nil};defer rows.Close();var o []Form;for rows.Next(){var e Form;rows.Scan(&e.ID,&e.Title,&e.Description,&e.Fields,&e.ResponseCount,&e.Status,&e.CreatedAt);o=append(o,e)};return o}
func(d *DB)Delete(id string)error{_,err:=d.db.Exec(`DELETE FROM forms WHERE id=?`,id);return err}
func(d *DB)Count()int{var n int;d.db.QueryRow(`SELECT COUNT(*) FROM forms`).Scan(&n);return n}
