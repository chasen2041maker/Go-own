package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "仅支持GET请求", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		name = "RUNNOOB"
	}

	resp := Response{
		Code:    200,
		Message: fmt.Sprintf("你好，%s！欢迎访问 Go HTTP 服务", name),
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	http.HandleFunc("/hello", helloHandler)

	addr := ":8080"
	fmt.Printf("RUNOOB 服务器启动，监听地址: http://localhost%s\n", addr)
	fmt.Printf("访问示例: http://localhost%s/hello?name=Go开发者\n", addr)

	log.Fatal(http.ListenAndServe(addr, nil))
}
