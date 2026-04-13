package main
import ("fmt";"log";"net/http";"os";"path/filepath";"github.com/stockyard-dev/stockyard-surveyor/internal/server";"github.com/stockyard-dev/stockyard-surveyor/internal/store";"github.com/stockyard-dev/stockyard/bus")
func main(){port:=os.Getenv("PORT");if port==""{port="9700"};dataDir:=os.Getenv("DATA_DIR");if dataDir==""{dataDir="./surveyor-data"}
db,err:=store.Open(dataDir);if err!=nil{log.Fatalf("surveyor: %v",err)};defer db.Close()
// Bus: one level up from private data dir so all tools in a bundle share
// one _bus.db. Non-fatal: surveyor serves users with or without it.
var b *bus.Bus
if bb,berr:=bus.Open(filepath.Dir(dataDir),"surveyor");berr!=nil{log.Printf("surveyor: bus disabled: %v",berr)}else{b=bb;defer b.Close()}
srv:=server.New(db,server.DefaultLimits(),b)
fmt.Printf("\n  Surveyor — Self-hosted form builder and survey tool\n  Dashboard:  http://localhost:%s/ui\n  API:        http://localhost:%s/api\n  Questions? hello@stockyard.dev — I read every message\n\n",port,port)
log.Printf("surveyor: listening on :%s",port);log.Fatal(http.ListenAndServe(":"+port,srv))}
