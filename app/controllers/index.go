package controllers

import (
	"encoding/json"
	"net/http"
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(struct {
		Status string		`json:"status"`
		Application string 	`json:"application"`
		Version float64 	`json:"version"`
		Author string 		`json:"author"`
		Email string 		`json:"email"`
		Company string 		`json:"company"`
		Owner string 		`json:"owner"`
	}{
		Status:  "Alive",
		Application: "PostIt Scheduler",
		Version: 1.0,
		Author:  "Prince Bobby",
		Email:   "princebobby506@gmail.com",
		Company: "Shiftr GH",
		Owner:   "Shiftr GH",
	})
}
