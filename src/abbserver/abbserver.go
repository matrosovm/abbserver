package abbserver

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

type (
	link [10]rune
 	localDB struct {
		sync.RWMutex
		LinkURL map[link]string
		URLLink map[string]link
	}
	postgresDB struct {
		*sql.DB
	}
)

var (
	possibleRunes [63]rune
	currentUniqLink [10]int8
	db interface {
		writeDB(url string, lk link) (ok bool)
		readLinkDB(lk link) (string, bool)
		readURLDB(url string) (link, bool)
	}
)

func init() {
	position := 0
	for i := 'A'; i <= 'Z'; i++ {
		possibleRunes[position] = i
		position++
	}
	for i := 'a'; i <= 'z'; i++ {
		possibleRunes[position] = i
		position++
	}
	for i := '0'; i <= '9'; i++ {
		possibleRunes[position] = i
		position++
	}
	possibleRunes[position] = '_'
	rand.Seed(time.Now().UnixMicro())
	rand.Shuffle(len(possibleRunes), func(i, j int) {
        possibleRunes[i], possibleRunes[j] = possibleRunes[j], possibleRunes[i]
	})

	for i := range currentUniqLink {
		currentUniqLink[i] = int8(rand.Intn(63))
	}
}

func generateNextUniqLink() link {
	for i := 9; i >= 0; i-- {
		currentUniqLink[i]++
		if currentUniqLink[i] < 63 {
			break
		}
		currentUniqLink[i] = 0
	}
	result := link{}
	for i, j := range currentUniqLink {
		result[i] = possibleRunes[j]
	}
	return result
}

func bytesToLink(str []byte) link {
	result := [10]rune{}
	copy(result[:], []rune(string(str)))
	return result
}

func (l *localDB) writeDB(url string, lk link) (ok bool) {
	l.Lock()
	defer l.Unlock()
	l.LinkURL[lk] = url
	l.URLLink[url] = lk
	return true
}

func (pDB *postgresDB) writeDB(url string, lk link) (ok bool) {
	sqlStatement := `INSERT INTO abbserver (url, link) VALUES ($1, $2)`
	_, err := pDB.Exec(sqlStatement, url, string(lk[:10]))
	if err != nil {
		fmt.Printf("writeDB: %s\n", err)
		return false
	}
	return true
}

func (l *localDB) readLinkDB(lk link) (string, bool) {
	l.RLock()
	defer l.RUnlock()
	url, ok := l.LinkURL[lk]
	return url, ok
}

func (pDB *postgresDB) readLinkDB(lk link) (string, bool) {
	sqlStatement := `SELECT url FROM abbserver WHERE link=$1`
	var url string
	err := pDB.QueryRow(sqlStatement, string(lk[:])).Scan(&url)
	switch err {
	case sql.ErrNoRows:
	case nil:
		return url, true
	default:
		fmt.Printf("readLinkDB: %s\n", err)
	}  
	return "", false
}

func (l *localDB) readURLDB(url string) (link, bool) {
	l.RLock()
	defer l.RUnlock()
	lk, ok := l.URLLink[url]
	return lk, ok
}

func (pDB *postgresDB) readURLDB(url string) (link, bool) {
	sqlStatement := `SELECT link FROM abbserver WHERE url=$1`
	var tmp string
	err := pDB.QueryRow(sqlStatement, url).Scan(&tmp)
	switch err {
	case sql.ErrNoRows:
	case nil:
		return bytesToLink([]byte(tmp)), true
	default:
		fmt.Printf("readURLDB: %s\n", err)
	}  
	return link{}, false
}

func post(w http.ResponseWriter, body []byte) {
	uniqLink, ok := db.readURLDB(string(body)) 
	if ok {
		fmt.Fprintf(w, "For url %s link is %s\n",  
		string(body), string(uniqLink[:]))
		w.WriteHeader(http.StatusOK)
		return
	}
	prevLink := uniqLink
	uniqLink = generateNextUniqLink()
	ok = db.writeDB(string(body), uniqLink)
	if ok {
		fmt.Fprintf(w, "For url %s link is %s\n", 
			string(body), string(uniqLink[:]))
		w.WriteHeader(http.StatusOK)
	} else {
		uniqLink = prevLink
		http.Error(w, "Error adding url in database", http.StatusBadRequest)
	}
}

func get(w http.ResponseWriter, body []byte) {
	url, ok := db.readLinkDB(bytesToLink(body))
	if !ok {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	} 
	fmt.Fprintf(w, "For link %s url is %s\n", string(body), string(url))
	w.WriteHeader(http.StatusOK)
}

func handler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Body reading error", http.StatusUnsupportedMediaType)
	}
	defer r.Body.Close()

	switch r.Method {
	case http.MethodPost:
		post(w, body)
	case http.MethodGet:
		get(w, body)
	default:
		http.Error(w, "Only POST and GET requests", http.StatusBadRequest)
	}
}

const (
	host = "localhost"
	port = 5432
	user = "postgres"
	password = "1234"
	mode = "disable"
)

func connectPostgres() *sql.DB {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
    "password=%s sslmode=%s",
    host, port, user, password , mode)
  	db, err := sql.Open("postgres", psqlInfo)
  	if err != nil {
    	panic(err)
  	}
	return db
}
  
func Connect(postgresMode bool) {
	if postgresMode {
		pDB := connectPostgres()
		defer pDB.Close()
		db = &postgresDB{pDB}
	} else {
		db = &localDB{LinkURL: make(map[link]string), 
			URLLink: map[string]link{}}
	}

	http.HandleFunc("/", handler) 
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Connect: ", err)
	}
}