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
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type Task struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

var db *sql.DB

func main() {
	var err error
	db, err = InitDB()
	if err != nil {
		log.Fatal(err)
	}

	if err := InitRouter(); err != nil {
		log.Fatal(err)
	}
	err = http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	if err != nil {
		panic(err)
	}
}

func InitRouter() error {
	http.HandleFunc("/tasks", GetAllTasks)
	http.HandleFunc("/task/create", CreateTask)
	http.HandleFunc("/tasks/delete/", DeleteTask)
	return nil
}

func GetAllTasks(w http.ResponseWriter, r *http.Request) {
	// タスクの取得
	rows, err := db.Query("SELECT id, title, description FROM tasks")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// 取得したタスクを格納するスライス
	tasks := []Task{}

	// 取得したタスクをスライスに格納する
	for rows.Next() {
		task := Task{}
		if err := rows.Scan(&task.ID, &task.Title, &task.Description); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, task)
	}

	// スライスをJSONに変換してレスポンスとして返す
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(tasks); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func CreateTask(w http.ResponseWriter, r *http.Request) {

	// リクエストボディからJSONデータをパースする
	var task Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		// エラー処理
	}

	now := time.Now().In(jst)
	// データベースにタスクを追加する
	stmt, err := db.Prepare("INSERT INTO tasks(title, description, created_at, updated_at) VALUES($1, $2, $3, $4)")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = stmt.Exec(task.Title, task.Description, now, now)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// レスポンスを返す
	w.WriteHeader(http.StatusCreated)
}

func DeleteTask(w http.ResponseWriter, r *http.Request) {
	// URLパラメータからタスクIDを取得する
	parts := strings.Split(r.URL.Path, "/")
	idStr := parts[len(parts)-1]
	if idStr == "" {
		http.Error(w, "id parameter is missing", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id parameter", http.StatusBadRequest)
		return
	}

	// タスクを削除する
	result, err := db.Exec("DELETE FROM tasks WHERE id = $1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 削除されたタスクの数を取得する
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, fmt.Sprintf("task with id %d not found", id), http.StatusNotFound)
		return
	}

	// レスポンスとして削除されたタスクのIDを返す
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%d", id)
}

func InitDB() (*sql.DB, error) {
	err := godotenv.Load()

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", os.Getenv("POSTGRES_HOST"), os.Getenv("POSTGRES_PORT"), os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PW"), os.Getenv("POSTGRES_DB"))
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	return db, nil
}
