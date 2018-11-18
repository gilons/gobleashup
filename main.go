package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	createUsersTable()
	var routes = mux.NewRouter()
	routes.HandleFunc("/test/createuser", CreateUser).Methods("POST")
	routes.HandleFunc("/test/login", Login).Methods("POST")
	routes.HandleFunc("/test/update", UpdateProfile).Methods("POST")
	routes.HandleFunc("/test/profileimage", SendProfilePhoto).Methods("GET")
	routes.HandleFunc("/test/createevent", CreateNewEvent).Methods("POST")
	routes.HandleFunc("/test/updateevent", UpdateEvent).Methods("POST")
	routes.HandleFunc("/test/addorganiser", AddEventOrganiser).Methods("POST")
	routes.HandleFunc("/test/addorganiserslave", AddEventOrganiserSlave).Methods("POST")
	routes.HandleFunc("/test/addconcerneeslave", AddConcerneeSlave).Methods("POST")

	http.Handle("/", routes)
	http.ListenAndServe(":8080", nil)
}

func createUsersTable() {
	db, err := initialiseDBSimple()
	if err != nil {
		fmt.Println(err.Error())
	}
	sqlQuery := "Create Table USERS (" +
		"user_id  varchar(256) not null PRIMARY KEY," +
		"user_phone varchar(256) NOT NULL UNIQUE," +
		"user_nickname varchar(32)," +
		"user_status varchar(256)," +
		"user_profile text," +
		"user_name varchar(256)," +
		"user_password text NOT NULL," +
		"user_salt varchar(256) not null )"
	result, err := (*db).Exec(sqlQuery)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(result)
}
