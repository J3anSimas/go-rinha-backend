package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type Pessoa struct {
	Id         string   `json:"id"`
	Apelido    string   `json:"apelido"`
	Nome       string   `json:"nome"`
	Nascimento string   `json:"nascimento"`
	Stack      []string `json:"stack"`
}

func main() {
	godotenv.Load(".env")
	db_password := os.Getenv("APP_DB_PASSWORD")
	db_host := os.Getenv("APP_DB_HOST")
	db_port := os.Getenv("APP_DB_PORT")
	db_username := os.Getenv("APP_DB_USER")
	db_name := os.Getenv("APP_DB_DB")
	strcon := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", db_username, db_password, db_host, db_port, db_name)
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		var p Pessoa
		err := json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if p.Apelido == "" || p.Nome == "" || p.Nascimento == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if len(p.Apelido) > 32 || len(p.Nome) > 100 {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return

		}
		splittedDate := strings.Split(p.Nascimento, "-")
		if len(splittedDate[0]) != 4 || len(splittedDate[1]) != 2 || len(splittedDate[2]) != 2 {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		_, err = strconv.Atoi(strings.Join(splittedDate, ""))
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		p.Id = uuid.New().String()
		fmt.Println(len(p.Id))
		conn, err := sql.Open("postgres", strcon)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		result, err := conn.Query("INSERT INTO pessoa (id, apelido, nome, nascimento, stack) VALUES ($1, $2, $3, $4, $5)", p.Id, p.Apelido, p.Nome, p.Nascimento, strings.Join(p.Stack, ","))
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		result.Close()
		w.Header().Add("Location", "/pessoas/"+p.Id)
		w.WriteHeader(http.StatusCreated)
	})
	r.Get("/pessoas/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		conn, err := sql.Open("postgres", strcon)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		result, err := conn.Query("SELECT * FROM pessoa WHERE id = $1", id)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var p Pessoa
		strStacks := ""
		for result.Next() {
			err := result.Scan(&p.Id, &p.Apelido, &p.Nome, &p.Nascimento, &strStacks)
			if err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		if p.Id == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		p.Stack = strings.Split(strStacks, ",")
		log.Println(p.Stack)
		w.Header().Add("Content-Type", "application/json")

		response, err := json.Marshal(p)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(response)

	})

	r.Get("/pessoas", func(w http.ResponseWriter, r *http.Request) {

		term := r.URL.Query().Get("t")
		conn, err := sql.Open("postgres", strcon)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var result *sql.Rows
		if term == "" {
			result, err = conn.Query("SELECT id, apelido, nome, nascimento, stack FROM pessoa limit 50")
		} else {
			result, err = conn.Query("SELECT id, apelido, nome, nascimento, stack FROM pessoa WHERE apelido LIKE $1 or nome LIKE $1 or stack LIKE $1 limit 50", "%"+term+"%")
		}
		var ps []Pessoa
		for result.Next() {
			strStacks := ""
			var p Pessoa
			err := result.Scan(&p.Id, &p.Apelido, &p.Nome, &p.Nascimento, &strStacks)
			if err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			p.Stack = strings.Split(strStacks, ",")
			ps = append(ps, p)
		}
		w.Header().Add("Content-Type", "application/json")
		if len(ps) == 0 {
			response, err := json.Marshal(make([]string, 0))
			if err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Write(response)
			return
		} else {
			response, err := json.Marshal(ps)
			if err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Write(response)
		}
	})
	r.Get("/contagem-pessoas", func(w http.ResponseWriter, r *http.Request) {
		conn, err := sql.Open("postgres", strcon)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		result, err := conn.Query("select count(*) from pessoa")
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var count int
		for result.Next() {
			err := result.Scan(&count)
			if err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		w.Write([]byte(strconv.Itoa(count)))

	})
	http.ListenAndServe(":3000", r)

}
