package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"

	"path/filepath"

	"github.com/gilons/apimaster/api"
	"github.com/gilons/apimaster/password"
)

const (
	folder = "profilephotos/"
)

//User struct
type User struct {
	UserName     string `json:"name"`
	UserNickName string `json:"nickname"`
	UserPhone    string `json:"phonenumber"`
	UserID       string `json:"userid"`
	UserPassword string `json:"password"`
	UserSalt     string `json:"salt"`
}

//ErrorMessage Struct
type ErrorMessage struct {
	ErrorTitle       string `json:"title"`
	ErrorDescription string `json:"errordescription"`
}

//CreateUserResponse struct
type CreateUserResponse struct {
	ResponseMessage string       `json:"message"`
	ErrorMessage    ErrorMessage `json:"error"`
}

//LoginResponse struct is used to response to a login request
type LoginResponse struct {
	ErrorMessage ErrorMessage `json:"error"`
	UserInfo     User         `json:"userinfo"`
}

//InternalError is a Function that Generate an internal.
//This function is mostly implemented for reporting database errors
func InternalError(w *http.ResponseWriter, Response CreateUserResponse) {
	Response.ErrorMessage.ErrorTitle = "INTERNAL ERROR"
	Response.ErrorMessage.ErrorDescription = "We are very sory !! An internal error occured" +
		" when trying to register you.Please Try again later while we are trying solve this problem"
	Response.ResponseMessage = ""
	res, err := json.Marshal(Response)
	if err != nil {
		fmt.Println(err)
	}
	(*w).Write(res)
}

//CreateUser is a function that initialise a new user in the database.
//This function writes back a http reply the user id so that it should be stored on the user's app interface.
func CreateUser(w http.ResponseWriter, r *http.Request) {
	UserInstance := User{}
	Response := CreateUserResponse{}
	UserInstance.UserName = r.FormValue("user_name")
	UserInstance.UserPhone = r.FormValue("phone_number")
	UserInstance.UserPassword = r.FormValue("password")
	UserInstance.UserNickName = r.FormValue("nick_name")
	db, err := api.MakeMsqlConnection("santers1997", "bleashup", "root")
	if err != nil {
		InternalError(&w, Response)
		fmt.Println(err.Error())
		return

	}
	salt, passwordhash := password.ReturnPassword(UserInstance.UserPassword)
	UserInstance.UserID = password.GenerateRandomID()
	UserInstance.UserID = strings.Replace(UserInstance.UserID, "-", "_", -1)
	queryString := "INSERT INTO `USERS` SET " +
		"`user_id` = \"" + UserInstance.UserID + "\"," +
		" `user_name` = \"" + UserInstance.UserName + "\" ," +
		"`user_nickname` = \"" + UserInstance.UserNickName + "\" ," +
		"`user_password` = \"" + passwordhash + "\" ," +
		"`user_phone` = \"" + UserInstance.UserPhone + "\" ," +
		"`user_salt` = \"" + salt + "\";"
		fmt.Println(queryString)
	result, err := db.Exec(queryString)
	if err != nil {
		mysqlErrorCode, errorMessage := GetMySQLErrorCode(err.Error())
		if mysqlErrorCode == 1062 && strings.
			Contains(errorMessage, "PRIMARY") && strings.Contains(errorMessage, "Duplicate") {
			CreateUser(w, r)
		} else if mysqlErrorCode == 1062 && strings.
			Contains(errorMessage, "Duplicate") {
			UserExits(&w, Response)
			fmt.Println(err.Error())
			return
		} else {
			InternalError(&w, Response)
			fmt.Println(err.Error())
			return
		}
	}
	fmt.Println(result)
	queryString = "CREATE TABLE `" + UserInstance.UserID + "_new_event` (" +
		"`invitor_id` varchar(256) NOT NULL , " +
		"`event_id` varchar(256) NOT NULL PRIMARY KEY, " +
		"`event_type` varchar(256) NOT NULL," +
		"`master_id` varchar(256) NOT NULL );"
	result, err = db.Exec(queryString)
	if err != nil {
		InternalError(&w, Response)
		fmt.Println(err.Error())
		return
	}
	fmt.Println(result)
	queryString = "CREATE TABLE `" + UserInstance.UserID + "_registered_event_master` (" +
		"`event_id` varchar(256) not null PRIMARY KEY, " +
		"`event_date` TIMESTAMP UNIQUE NOT NULL ," +
		"`event_place` VARCHAR(256) NOT NULL," +
		"`event_description` TEXT ," +
		"`event_concernee` TEXT ," +
		"`event_organisers` TEXT );"
		fmt.Println(queryString)
	result, err = db.Exec(queryString)
	if err != nil {
		InternalError(&w, Response)
		fmt.Println(err.Error())
		return
	}
	fmt.Println(result)
	queryString = "CREATE TABLE `" + UserInstance.UserID + "_register_event_slave` (" +
		"`master_id` varchar(256) UNIQUE NOT NULL ," +
		"`event_id` varchar(256)  NOT NULL PRIMARY KEY, " +
		"`invitor_id` varchar(256) UNIQUE NOT NULL );"
	result, err = db.Exec(queryString)
	if err != nil {
		fmt.Println(err.Error())
		InternalError(&w, Response)
		return
	}
	fmt.Println(result)
	Response.ErrorMessage = ErrorMessage{ErrorTitle: "", ErrorDescription: ""}
	Response.ResponseMessage = UserInstance.UserID
	res, err := json.Marshal(Response)
	if err != nil {
		fmt.Println(err)
	}
	w.Write(res)
}

func initialiseDB(w *http.ResponseWriter, Response CreateUserResponse) (api.Dbase, error) {
	db, err := api.MakeMsqlConnection("santers1997", "bleashup", "root")
	if err != nil {
		InternalError(w, Response)
		fmt.Println(err.Error())
		return nil, err
	}
	return db, nil
}

func initialiseDBSimple() (api.Dbase, error) {
	db, err := api.MakeMsqlConnection("santers1997", "bleashup", "root")
	if err != nil {
		return nil, err
	}
	return db, nil
}

//Login is a handler funct that is called when there is a login event
func Login(w http.ResponseWriter, r *http.Request) {
	var loginUser, realUser User
	response := LoginResponse{}
	createResponse := CreateUserResponse{}
	db, err := initialiseDB(&w, createResponse)
	if err != nil {
		return
	}
	loginUser.UserPhone = r.FormValue("phone_number")
	loginUser.UserPassword = r.FormValue("password")
	sqlQuery := "select user_password,user_salt from USERS where user_phone = " + loginUser.UserPhone
	err = (*db).QueryRow(sqlQuery).Scan(&realUser.UserPassword, &realUser.UserSalt)
	if err != nil || realUser.UserPassword == "" {
		fmt.Println(err.Error())
		ErrornousInFo(&w, response)
		return
	}

	passwordhash := password.GenerateHash(realUser.UserSalt, loginUser.UserPassword)

	fmt.Println(realUser.UserPassword, passwordhash, "*********")
	if passwordhash == realUser.UserPassword {
		var temp1, temp2, temp3 string
		sqlQuery = "select * from USERS where user_phone = " + loginUser.UserPhone
		err = (*db).QueryRow(sqlQuery).Scan(&response.UserInfo.UserID, &response.UserInfo.UserPhone,
			&response.UserInfo.UserNickName, &temp1, &temp2, &response.UserInfo.UserName,
			&response.UserInfo.UserPassword, &temp3)
		if err != nil {
			fmt.Println(err)
		}
		sqlQuery = "select user_name from USERS where user_phone = " + loginUser.UserPhone
		err = (*db).QueryRow(sqlQuery).Scan(&response.UserInfo.UserName)
		if err != nil {
			fmt.Println(err)
		}
		response.UserInfo.UserPassword = ""
		response.ErrorMessage = ErrorMessage{}
		jsonresp, err := json.Marshal(response)
		if err != nil {
			fmt.Println(err.Error())
		}
		w.Write(jsonresp)
		return
	}
	WrongPassword(&w, response)
	return

}

//WrongPassword is a function tha generates the corresponding error message for Wrong password entry
func WrongPassword(w *http.ResponseWriter, Response LoginResponse) {
	Response.ErrorMessage.ErrorTitle = "WRONG PASSWORD"
	Response.ErrorMessage.ErrorDescription = "We are sory .We are unable to login deu to a wrong password." +
		"Please check You Password , Ensure that They it is Correct and try again"
	Response.UserInfo = User{}
	res, err := json.Marshal(Response)
	if err != nil {
		fmt.Println(err)
	}
	(*w).Write(res)
}

//ErrornousInFo is a function that Generates an error message for an Caused to Erronous Entry from the User
func ErrornousInFo(w *http.ResponseWriter, Response LoginResponse) {
	Response.ErrorMessage.ErrorTitle = "WRONG ENTRY"
	Response.ErrorMessage.ErrorDescription = "We are sory .We are unable to login deu to a wrong entry." +
		"Please check You Info , Ensure that They are Correct and try again"
	Response.UserInfo = User{}
	res, err := json.Marshal(Response)
	if err != nil {
		fmt.Println(err)
	}
	(*w).Write(res)
}

//UserExits is a functin that reports the error about a userthat already exist
func UserExits(w *http.ResponseWriter, Response CreateUserResponse) {
	Response.ErrorMessage.ErrorTitle = "USER ALREADY EXIST"
	Response.ErrorMessage.ErrorDescription = "there already exist an account" +
		" register with this Phone number"
	Response.ResponseMessage = ""
	res, err := json.Marshal(Response)
	if err != nil {
		fmt.Println(err.Error())
	}
	(*w).Write(res)
}

//GetMySQLErrorCode is a function that gives out the corresponding mysql error string together with error code
func GetMySQLErrorCode(err string) (int64, string) {
	parts := strings.Split(err, ":")
	errorMessage := parts[1]
	code := strings.Split(parts[0], "Error ")
	errorCode, _ := strconv.ParseInt(code[1], 10, 32)
	return errorCode, errorMessage
}

//UpdateProfile function is a function that Updates UserInfo in the database
func UpdateProfile(w http.ResponseWriter, r *http.Request) {
	whatToDo := r.FormValue("whattodo")
	phoneNumber := r.FormValue("phonenumber")
	newInfo := r.FormValue("newinfo")
	db, err := initialiseDBSimple()
	if err != nil {
		fmt.Println(err.Error())
	}
	Rows := make(map[string]string)
	Rows["status"] = "user_status"
	Rows["name"] = "user_name"
	Rows["nickname"] = "user_nickname"
	Rows["phone"] = "user_phone"
	if whatToDo == "profile" {
		err := savePhoto(w, r)
		if err != nil {
			fmt.Println(err.Error())
		}
	} else {
		updateUserQuery(phoneNumber, Rows[whatToDo], newInfo, db)
	}
	w.Write(nil)
}

func retrievePhotoInfo(w http.ResponseWriter, r *http.Request) (file, error) {
	fileInstance := file{}
	var err error
	if err = r.ParseMultipartForm(5 * MB); err != nil {
		fmt.Println(err.Error())
		return file{}, err
	}
	fileInstance.File, fileInstance.FileHeader, err = r.FormFile("newinfo")
	fmt.Println(fileInstance.FileHeader.Filename, "********************")
	if err != nil {
		fmt.Println(err.Error())
		return file{}, err
	}
	return fileInstance, nil

}

const (
	//MB represents a megabyte
	MB = 1 << 20
)

type file struct {
	File       multipart.File
	FileHeader *multipart.FileHeader
}

func savePhoto(w http.ResponseWriter, r *http.Request) error {
	fileInstance := file{}
	phoneNumber := r.FormValue("phonenumber")
	fileInstance, err := retrievePhotoInfo(w, r)
	if err != nil {
		return err
	}
	defer fileInstance.File.Close()
	db, _ := initialiseDBSimple()
	var userID string
	sqlquery := "SELECT `user_id` FROM `USERS` WHERE `user_phone` = \"" + phoneNumber + "\""
	err = (*db).QueryRow(sqlquery).Scan(&userID)
	if err != nil {
		log.Println(err.Error())
	}

	filenames := strings.Split(fileInstance.FileHeader.Filename, ".")
	var file *os.File
	images, err := filepath.Glob(folder + userID + ".*")
	if err != nil {
		fmt.Println(err.Error())
	}
	if images != nil {
		_ = os.RemoveAll(folder + userID + "." + strings.Split(images[0], ".")[1])
		file, err = os.Create(folder + userID + "." + filenames[1])
		if err != nil {
			fmt.Println(err.Error())
		}
	} else {
		file, err = os.Create(folder + userID + "." + filenames[1])
		if err != nil {
			fmt.Println(err.Error())
		}
	}
	io.Copy(file, fileInstance.File)
	completDir := folder + userID + "." + filenames[1]
	sqlquery = "UPDATE  USERS SET `user_profile` = \"" + completDir + "\" WHERE `user_phone` = \"" + phoneNumber + "\""
	result, err := (*db).Exec(sqlquery)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(result)
	return nil
}

//SendProfilePhoto function is a function that responds to the /profileimage Endpoint
//It fetches and send the coresponding user's profile Photo
func SendProfilePhoto(w http.ResponseWriter, r *http.Request) {
	userPhone := r.FormValue("phone")
	var profileDire string
	db, err := initialiseDBSimple()
	if err != nil {
		fmt.Println(err.Error())
	}
	sqlQuery := "SELECT `user_profile` FROM `USERS` WHERE `user_phone` = " + userPhone
	err = (*db).QueryRow(sqlQuery).Scan(&profileDire)
	if err != nil {
		fmt.Println(err.Error())
	}
	profileImagePath, err := filepath.Abs(profileDire)
	if err != nil {
		fmt.Println(err.Error())
	}
	imageFile, err := ioutil.ReadFile(profileImagePath)
	if err != nil {
		fmt.Println(err.Error())
	}
	profileDireSlice := strings.Split(profileDire, ".")
	imageMemory := bytes.NewBuffer(imageFile)
	w.Header().Set("content-type", "image/"+profileDireSlice[1])
	if _, err := imageMemory.WriteTo(w); err != nil {
		fmt.Fprintf(w, "something went wrong during the read operation")
	}
}

func updateUserQuery(userPhone string, row string, newData string, db *sql.DB) {
	mysqlQueryString := "UPDATE `USERS` SET `" + row + "` = \"" +
		newData + "\" where `user_phone` = \"" + userPhone + "\""
	result, err := (*db).Exec(mysqlQueryString)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(result)

}
