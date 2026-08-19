package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

var (
	users  = make(map[int]User)
	nextID = 1
	mu     sync.RWMutex
)

func task1_helloHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	fmt.Fprintf(w, "Hello, World!")
}

func task2_getUsersHandler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()

	userList := make([]User, 0, len(users))
	for _, user := range users {
		userList = append(userList, user)
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(userList); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func task3_createUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	mu.Lock()
	user.ID = nextID
	nextID++
	users[user.ID] = user
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s", r.Method, r.URL.Path)

		// 调用下一个处理器
		next(w, r)
	}
}

func task4_setupRoutes() {
	http.HandleFunc("/hello", task1_helloHandler)

	http.HandleFunc("/users", loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// 根据 HTTP 方法路由到不同的处理器
		switch r.Method {
		case http.MethodGet:
			task2_getUsersHandler(w, r)
		case http.MethodPost:
			task3_createUserHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
}
