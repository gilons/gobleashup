package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gilons/apimaster/password"
)

//CreateNewEvent is a function tha resqponse to the endpoint createevent .
//It permits the user to Cerate a new event a master
func CreateNewEvent(w http.ResponseWriter, r *http.Request) {
	time := r.FormValue("time")
	date := r.FormValue("date")
	eventDescription := r.FormValue("description")
	//The concernees carries the concernees phone number in a single
	//string with the phone numbers separated by commas
	//eg concernees data may look like 654656464,6546564654,65654654646,6546565456
	concernees := r.FormValue("concernees")
	eventPlace := r.FormValue("location")
	userPhone := r.FormValue("phone")
	eventType := r.FormValue("eventtype")
	//storing the event_info in the database
	db, err := initialiseDBSimple()
	if err != nil {
		fmt.Println(err.Error())
	}
	userID := GetUserID(userPhone, db)
	eventID := password.GenerateRandomID()
	eventID = strings.Replace(eventID, "-", "_", -1)
	mysqlQueryString := "INSERT INTO `" + userID + "_registered_event_master` SET " +
		"`event_id` = \"" + eventID + "\", " +
		"`event_date` = \"" + date + " " + time + "\"" +
		", `event_description` = \"" + eventDescription + "\"" +
		", `event_concernee` = \"" + concernees + "\"" +
		", `event_place` = \"" + eventPlace + "\""
	result, err := (*db).Exec(mysqlQueryString)
	if err != nil {
		mysqlErrorCode, errorMessage := GetMySQLErrorCode(err.Error())
		if mysqlErrorCode == 1062 && strings.
			Contains(errorMessage, "PRIMARY") && strings.Contains(errorMessage, "Duplicate") {
			CreateNewEvent(w, r)
		} else if mysqlErrorCode == 1062 && strings.
			Contains(errorMessage, "Duplicate") {
			response := ErrorMessage{}
			duplicateEventTime(&w, response)
			fmt.Println(err.Error())
			return
		} else {
			fmt.Println(err.Error())
		}
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(rowsAffected)
	w.Write([]byte(eventID))
	concerneesArray := strings.Split(concernees, ",")
	InformConcerneesAsMaster(userID, eventID, concerneesArray, eventType, db)

}
func duplicateEventTime(w *http.ResponseWriter, response ErrorMessage) {
	response.ErrorTitle = "Duplicate Time"
	response.ErrorDescription = "another event is already set For this period " +
		"Please ensure that the date of time are different and revalidate your event"
	result, _ := json.Marshal(response)
	(*w).Write(result)
}

//GetUserID is a Functin that gets the UserID of a Specific user given his phone number
func GetUserID(userPhone string, db *sql.DB) string {
	var userID string
	mysqlQueryString := "SELECT `user_id` FROM `USERS` WHERE `user_phone` = \"" + userPhone + "\""
	err := db.QueryRow(mysqlQueryString).Scan(&userID)
	if err != nil {
		fmt.Println(err.Error())
	}
	return userID
}

//UpdateEvent is a function that response to the endpoint updateevent
//It permits the user to change information about an event a new change should shown to all the
//concernees of the event.
func UpdateEvent(w http.ResponseWriter, r *http.Request) {
	userPhone := r.FormValue("phone")
	whatToDo := r.FormValue("whattodo")
	eventID := r.FormValue("eventid")
	newData := r.FormValue("newdata")
	eventType := r.FormValue("eventtype")
	Rows := make(map[string]string)
	Rows["place"] = "event_place"
	Rows["time"] = "event_time"
	Rows["date"] = "event_date"
	Rows["description"] = "event_description"
	Rows["concernee"] = "event_concernee"
	db, err := initialiseDBSimple()
	if err != nil {
		fmt.Println(err.Error())
	}
	userID := GetUserID(userPhone, db)
	if whatToDo == "concernee" {
		eventConcernee, err := selectFromRegisteredEventMaster(userID, "event_concernee", eventID, db)
		if err != nil {
			fmt.Println(err.Error())
		}
		newDataArray := strings.Split(newData, ",")
		eventConcerneeArray := strings.Split(eventConcernee, ",")
		status, exixtingConcernee := testIfEachElementExists(newDataArray,
			eventConcerneeArray)
		if status == true {
			ErrorResponse := ErrorMessage{}
			duplicateConcerneeError(&w, ErrorResponse, exixtingConcernee)
			return
		}
		var concernee string
		if eventConcernee == "" {
			concernee = newData
		} else {
			concernee = eventConcernee + "," + newData
		}

		updateEventQuery(userID, eventID, Rows[whatToDo], concernee, db)
		w.Write([]byte("OK"))
		InformConcerneesAsMaster(userID, eventID, newDataArray, eventType, db)
		return
	}
	updateEventQuery(userID, eventID, Rows[whatToDo], newData, db)
	w.Write([]byte("OK"))

}

//AddEventOrganiser func isa function that listen to the addorganiser endpoint
func AddEventOrganiser(w http.ResponseWriter, r *http.Request) {
	masterPhone := r.FormValue("phone")
	eventID := r.FormValue("eventid")
	newOrganisers := r.FormValue("neworganisers")
	db, err := initialiseDBSimple()
	if err != nil {
		fmt.Println(err.Error())
	}
	masterID := GetUserID(masterPhone, db)
	eventOrganisers, err := selectFromRegisteredEventMaster(masterID,
		"event_organisers", eventID, db)
	if err != nil {
		fmt.Println(err.Error())
	}
	newOrganisersArray := strings.Split(newOrganisers, ",")
	eventOrganisersArray := strings.Split(eventOrganisers, ",")
	status, organiser := testIfEachElementExists(newOrganisersArray, eventOrganisersArray)
	if status == true {
		errorResponse := ErrorMessage{}
		duplicateOrganiserError(&w, errorResponse, organiser)
	} else {
		var TotalOrganisers string
		if eventOrganisers == "" {
			TotalOrganisers = newOrganisers
		} else {
			TotalOrganisers = newOrganisers + "," + eventOrganisers
		}
		updateEventQuery(masterID, eventID, "event_organisers", TotalOrganisers, db)
	}
	w.Write([]byte("OK"))

}

func selectFromRegisteredEventMaster(masterID, row, eventID string, db *sql.DB) (string, error) {
	mySQLQueryString := "SELECT `" + row + "` FROM `" + masterID +
		"_registered_event_master` WHERE `event_id` = \"" + eventID + "\""
	var data string
	err := (*db).QueryRow(mySQLQueryString).Scan(&data)
	if err != nil {
		return "", err
	}
	return data, nil
}

//testIfEachElementExists tests each element of an array to see if any one of them exists in another array
func testIfEachElementExists(newArray []string, oldArray []string) (bool, string) {
	for _, element := range newArray {
		if status := Contains(oldArray, element); status == true {
			return true, element
		}
	}
	return false, ""
}

func duplicateConcerneeError(w *http.ResponseWriter, response ErrorMessage, concernee string) {
	response.ErrorTitle = "Duplicate Concernee"
	response.ErrorDescription = "The Concernee with phone number " + concernee + "is already aware of this event"
	resp, _ := json.Marshal(response)
	(*w).Write(resp)
}

func duplicateOrganiserError(w *http.ResponseWriter, response ErrorMessage, organiser string) {
	response.ErrorTitle = "Duplicate Organiser"
	response.ErrorDescription = "The Organiser with phone number " + organiser + "is already among the event organiser"
	resp, _ := json.Marshal(response)
	(*w).Write(resp)
}

//updateEventQuery is a function that perfor the corresponding update operation.
func updateEventQuery(UserID string, eventID string, row string, newData string, db *sql.DB) {
	mysqlQueryString := "UPDATE `" + UserID + "_registered_event_master` SET `" + row + "` = \"" +
		newData + "\" where `event_id` = \"" + eventID + "\""
	result, err := (*db).Exec(mysqlQueryString)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(result)

}

//InformConcerneesAsMaster is a function that adds other concernee to the newly added event as Master
func InformConcerneesAsMaster(masterID string, eventID string, concernees []string, eventType string, db *sql.DB) {
	for _, concernee := range concernees {
		informSingleConcernee(masterID, masterID, eventID, concernee, eventType, db)
	}
}

func informSingleConcernee(invitorID string, masterID string, eventID string, concernee string, eventType string, db *sql.DB) {
	inviteeID := GetUserID(concernee, db)
	mySQLQueryString := "INSERT INTO `" + inviteeID + "_new_event` SET " +
		"`event_type` = \"" + eventType + "\"," +
		"`event_id` = \"" + eventID + "\"," +
		"`invitor_id` = \"" + invitorID + "\"," +
		"`master_id` = \"" + masterID + "\""
	result, err := (*db).Exec(mySQLQueryString)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(result)
}

//InformConcerneesAsSlave is a function that adds other concernee to the newly added event as Slave
func InformConcerneesAsSlave(eventID string, invitorPhone string, eventType string, concernees []string, db *sql.DB) {
	invitorID := GetUserID(invitorPhone, db)
	masterID, err := selectFromRegisteredEventSlave("master_id", invitorID, eventID, db)
	if err != nil {
		fmt.Println(err.Error())
	}
	for _, concernee := range concernees {
		informSingleConcernee(invitorID, masterID, eventID, concernee, eventType, db)
	}

}

//AddConcerneeSlave is a function that listen to the end point addconcerneeslave.
//this endpoint permits someone who was added to an event to add another one to that
//same event
func AddConcerneeSlave(w http.ResponseWriter, r *http.Request) {
	invitorPhone := r.FormValue("invitorPhone")
	eventID := r.FormValue("eventid")
	newConcernees := r.FormValue("concernee")
	eventType := r.FormValue("eventtype")
	db, err := initialiseDBSimple()
	if err != nil {
		fmt.Println(err.Error())
	}
	invitorID := GetUserID(invitorPhone, db)
	masterID, err := selectFromRegisteredEventSlave("master_id", invitorID, eventID, db)
	if err != nil {
		fmt.Println(err.Error())
	}
	organisers, err := selectFromRegisteredEventMaster(masterID, "event_organsisers", eventID, db)
	if err != nil {
		fmt.Println(err.Error())
	}
	if status := Contains(strings.Split(organisers, ","), invitorPhone); status == true {
		concernees, err := selectFromRegisteredEventMaster(masterID, "event_concernee", eventID, db)
		if err != nil {
			fmt.Println(err.Error())
		}
		status, concernee := testIfEachElementExists(strings.Split(newConcernees,
			","), strings.Split(concernees, ","))
		if status == true {
			errorResponse := ErrorMessage{}
			duplicateConcerneeError(&w, errorResponse, concernee)
			return
		}
		var totalConcernees string
		if concernee == "" {
			totalConcernees = newConcernees
		} else {
			totalConcernees = concernees + "," + newConcernees
		}
		updateEventQuery(masterID, eventID, "event_concernee", totalConcernees, db)
		w.Write([]byte("OK"))
		InformConcerneesAsSlave(eventID, invitorPhone, eventType, strings.Split(newConcernees, ","), db)
		return
	}
	responseMessage := "Sory but you'r not illigible to add new  participatants to this event"
	w.Write([]byte(responseMessage))
}

//AddEventOrganiserSlave Function is a function that listern to the endpoint addeventslave.
//itpermits the user who where invited or informed about an event and added a organiser
// to add another organiser of the event
func AddEventOrganiserSlave(w http.ResponseWriter, r *http.Request) {
	eventID := r.FormValue("eventid")
	adderPhone := r.FormValue("phone")
	newOrganisers := r.FormValue("organisers")
	db, err := initialiseDBSimple()
	if err != nil {
		fmt.Println(err.Error())
	}
	adderID := GetUserID(adderPhone, db)
	masterID, err := selectFromRegisteredEventSlave("master_id", adderID, eventID, db)
	if err != nil {
		fmt.Println(err.Error())
	}
	eventOrganisers, err := selectFromRegisteredEventMaster(masterID,
		"event_organisers", eventID, db)
	if err != nil {
		fmt.Println(err.Error())
	}
	if status := Contains(strings.Split(eventOrganisers, ","), adderPhone); status == true {
		status, organiser := testIfEachElementExists(strings.Split(newOrganisers, ","),
			strings.Split(eventOrganisers, ","))
		if status == true {
			newResponse := ErrorMessage{}
			duplicateOrganiserError(&w, newResponse, organiser)
			return
		}
		totalOrganisers := eventOrganisers + "," + newOrganisers
		updateEventQuery(masterID, eventID, "event_organisers", totalOrganisers, db)
		w.Write([]byte("OK"))
		return
	}
	responseMessage := "sory but you are not allowed to add new ornganisers to this event"
	w.Write([]byte(responseMessage))

}

func selectFromRegisteredEventSlave(row, invitorID, eventID string, db *sql.DB) (string, error) {
	var data string
	mySQLQueryString := "SELECT `" + row + "` FROM `" + invitorID + "_registered_event_slave` WHERE `event_id` = \"" + eventID + "\""
	err := (*db).QueryRow(mySQLQueryString).Scan(&data)
	if err != nil {
		return "", nil
	}
	return data, nil
}

// Contains tells whether a contains x.
func Contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}
