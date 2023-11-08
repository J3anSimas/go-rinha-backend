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
	// godotenv.Load(".env")
	app_port := os.Getenv("APP_PORT")
	db_username := os.Getenv("APP_DB_USERNAME")
	db_password := os.Getenv("APP_DB_PASSWORD")
	db_name := os.Getenv("APP_DB_NAME")
	db_host := os.Getenv("APP_DB_HOST")
	db_port := os.Getenv("APP_DB_PORT")
	strcon := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", db_username, db_password, db_host, db_port, db_name)
	fmt.Println(strcon)
	fmt.Println("Porta: " + app_port)
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Post("/pessoas", func(w http.ResponseWriter, r *http.Request) {
		conn, err := sql.Open("postgres", strcon)
		if err != nil {
			conn.Query("INSERT INTO LOG (MESSAGE) VALUES ($1)", "Erro ao conectar ao banco de dados (POST /pessoas")
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer conn.Close()
		var p Pessoa
		err = json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			conn.Query("INSERT INTO LOG (MESSAGE) VALUES ($1)", "Erro ao decodificar JSON (POST /pessoas")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if p.Apelido == "" || p.Nome == "" || p.Nascimento == "" {
			conn.Query("INSERT INTO LOG (MESSAGE, apelido, nome, nascimento) VALUES ($1, $2, $3, $4)", "Dados vazios (POST /pessoas", p.Apelido, p.Nome, p.Nascimento)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if len(p.Apelido) > 32 || len(p.Nome) > 100 {
			conn.Query("INSERT INTO LOG (MESSAGE, apelido, nome, nascimento) VALUES ($1, $2, $3, $4)", "Dados muito grandes (POST /pessoas", p.Apelido, p.Nome, p.Nascimento)
			w.WriteHeader(http.StatusUnprocessableEntity)
			return

		}
		splittedDate := strings.Split(p.Nascimento, "-")
		if len(splittedDate[0]) != 4 || len(splittedDate[1]) != 2 || len(splittedDate[2]) != 2 {
			conn.Query("INSERT INTO LOG (MESSAGE, apelido, nome, nascimento) VALUES ($1, $2, $3, $4)", "Data inválida (POST /pessoas", p.Apelido, p.Nome, p.Nascimento)
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		_, err = strconv.Atoi(strings.Join(splittedDate, ""))
		if err != nil {
			conn.Query("INSERT INTO LOG (MESSAGE, apelido, nome, nascimento) VALUES ($1, $2, $3, $4)", "Data inválida (POST /pessoas: "+err.Error(), p.Apelido, p.Nome, p.Nascimento)
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		p.Id = uuid.New().String()
		fmt.Println(len(p.Id))
		result, err := conn.Query("INSERT INTO pessoa (id, apelido, nome, nascimento, stack) VALUES ($1, $2, $3, $4, $5)", p.Id, p.Apelido, p.Nome, p.Nascimento, strings.Join(p.Stack, ","))
		if err != nil {
			conn.Query("INSERT INTO LOG (MESSAGE, apelido, nome, nascimento) VALUES ($1, $2, $3, $4)", "Erro ao inserir no banco de dados (POST /pessoas: "+err.Error(), p.Apelido, p.Nome, p.Nascimento)
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		result.Close()
		w.Header().Add("Location", "/pessoas/"+p.Id)
		w.WriteHeader(http.StatusCreated)
		return
	})
	r.Get("/pessoas/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		conn, err := sql.Open("postgres", strcon)
		defer conn.Close()
		if err != nil {
			conn.Query("INSERT INTO LOG (MESSAGE) VALUES ($1)", "Erro ao conectar ao banco de dados (GET /pessoas/"+id)
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		result, err := conn.Query("SELECT * FROM pessoa WHERE id = $1", id)
		if err != nil {
			conn.Query("INSERT INTO LOG (MESSAGE) VALUES ($1)", "Erro ao consultar banco de dados (GET /pessoas/"+id)
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var p Pessoa
		strStacks := ""
		for result.Next() {
			err := result.Scan(&p.Id, &p.Apelido, &p.Nome, &p.Nascimento, &strStacks)
			if err != nil {
				conn.Query("INSERT INTO LOG (MESSAGE) VALUES ($1)", "Erro ao decodificar JSON (GET /pessoas/"+id)
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		if p.Id == "" {
			conn.Query("INSERT INTO LOG (MESSAGE) VALUES ($1)", "Pessoa não encontrada (GET /pessoas/"+id)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		p.Nascimento = formatDate(p.Nascimento)
		p.Stack = strings.Split(strStacks, ",")
		w.Header().Add("Content-Type", "application/json")

		response, err := json.Marshal(p)
		if err != nil {
			conn.Query("INSERT INTO LOG (MESSAGE) VALUES ($1)", "Erro ao decodificar JSON (GET /pessoas/"+id)
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(response)

	})

	r.Get("/pessoas", func(w http.ResponseWriter, r *http.Request) {

		conn, err := sql.Open("postgres", strcon)
		defer conn.Close()
		if err != nil {
			conn.Query("INSERT INTO LOG (MESSAGE) VALUES ($1)", "Erro ao conectar ao banco de dados (GET /pessoas: "+err.Error())
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		term := r.URL.Query().Get("t")
		if term == "" {
			conn.Query("INSERT INTO  LOG (MESSAGE) VALUES ($1)", "Termo de busca vazio (GET /pessoas")
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
		}
		var result *sql.Rows

		result, err = conn.Query("SELECT id, apelido, nome, nascimento, stack FROM pessoa WHERE apelido LIKE $1 or nome LIKE $1 or stack LIKE $1 limit 50", "%"+term+"%")

		if err != nil {
			conn.Query("INSERT INTO LOG (MESSAGE) VALUES ($1)", "Erro ao consultar banco de dados (GET /pessoas: "+err.Error())
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var ps []Pessoa
		for result.Next() {
			strStacks := ""
			var p Pessoa
			err := result.Scan(&p.Id, &p.Apelido, &p.Nome, &p.Nascimento, &strStacks)
			if err != nil {
				conn.Query("INSERT INTO LOG (MESSAGE) VALUES ($1)", "Erro ao decodificar JSON (GET /pessoas: "+err.Error())
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			p.Stack = strings.Split(strStacks, ",")
			p.Nascimento = formatDate(p.Nascimento)
			ps = append(ps, p)
		}
		w.Header().Add("Content-Type", "application/json")
		if len(ps) == 0 {
			response, err := json.Marshal(make([]string, 0))
			if err != nil {
				conn.Query("INSERT INTO LOG (MESSAGE) VALUES ($1)", "Erro ao decodificar JSON (GET /pessoas: "+err.Error())
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Write(response)
			return
		} else {
			response, err := json.Marshal(ps)
			if err != nil {
				conn.Query("INSERT INTO LOG (MESSAGE) VALUES ($1)", "Erro ao decodificar JSON (GET /pessoas: "+err.Error())
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Write(response)
		}
	})
	r.Get("/contagem-pessoas", func(w http.ResponseWriter, r *http.Request) {
		conn, err := sql.Open("postgres", strcon)
		defer conn.Close()
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
	http.ListenAndServe(":"+app_port, r)

}

func formatDate(date string) string {
	// return fmt.Sprintf("%d-%d-%d", date.Year, date.Month, date.Day)
	return strings.Split(date, "T")[0]
}
