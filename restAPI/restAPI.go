package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

var DiscoveryIP string = "localhost"
var DSMasterIP string = ""
var DSBalancerIP string = ""

func reportDSMasterCrash() {
	var request string = DSMasterIP //Build the request in a particular format
	requestJSON, _ := json.Marshal(request)
	http.Post("http://"+DiscoveryIP+"/dsMasterCrash", "application/json", bytes.NewBuffer(requestJSON)) //Submitting a put request

}
func changeDSMasterOnCrash(w http.ResponseWriter, r *http.Request) {
	DSMasterIP = analyzeRequest(r)
}

func get(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "Application/json")
	params := mux.Vars(r)                                                              //Acquire url params
	response, err := http.Get("http://" + DSBalancerIP + ":8000/get/" + params["key"]) //Submitting a get request
	if err != nil {
		fmt.Println("An error has occurred trying to estabilish a connection with the API.")
		fmt.Println(err.Error())
		return
	}
	responseFromDS, err := ioutil.ReadAll(response.Body) //Receiving http response
	if err != nil {
		fmt.Println("An error has occurred trying to read the requested file.")
		fmt.Println(err.Error())
		return
	}
	fmt.Println(string(responseFromDS))
	json.NewEncoder(w).Encode(string(responseFromDS)) //Send the response to the client
}

func put(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "Application/json")
	receivedRequest := analyzeRequest(r)
	var info []string = strings.Split(receivedRequest, "|") //Acquire file name and content from client's request
	var fileName string = info[0]
	var fileContent string = info[1]
	var request string = fileName + "|" + fileContent //Build the request in a particular format
	requestJSON, _ := json.Marshal(request)
	_, err := http.Post("http://"+DSMasterIP+"/put", "application/json", bytes.NewBuffer(requestJSON)) //Submitting a put request
	if err != nil {
		reportDSMasterCrash()
		return
	}
	json.NewEncoder(w).Encode("The file was successfully uploaded.")
}

func del(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "Application/json")
	fileToRemove := analyzeRequest(r)
	var request string = fileToRemove //Build the request in a particular format
	requestJSON, _ := json.Marshal(request)
	_, err := http.Post("http://"+DSMasterIP+"/del", "application/json", bytes.NewBuffer(requestJSON)) //Submitting a put request
	if err != nil {
		reportDSMasterCrash()
		return
	}
	json.NewEncoder(w).Encode("The file was successfully removed.")
}

func main() {
	register()
	router := mux.NewRouter()                           //Router initialization
	router.HandleFunc("/put", put).Methods("POST")      //put requests handler/endpoint
	router.HandleFunc("/get/{key}", get).Methods("GET") //get requests handler/endpoint
	router.HandleFunc("/delete", del).Methods("POST")   //del requests handler/endpoint
	router.HandleFunc("/changeMaster", changeDSMasterOnCrash).Methods("POST")
	log.Fatal(http.ListenAndServe(":8000", router)) //Listen and serve requests on port 8000
}
func register() {
	requestJSON, _ := json.Marshal("restAPI")
	response, err := http.Post("http://"+DiscoveryIP+":8000/register", "application/json", bytes.NewBuffer(requestJSON))
	for err != nil { //Se fallisce riprova ogni 3 secondi
		fmt.Println("An error has occurred trying to estabilish a connection with the Discovery node.")
		fmt.Println(err.Error())
		time.Sleep(3 * time.Second)
		response, err = http.Post("http://"+DiscoveryIP+":8000/register", "application/json", bytes.NewBuffer(requestJSON))
	}
	responseFromDiscovery, _ := ioutil.ReadAll(response.Body) //Receiving http response
	if strings.Contains(string(responseFromDiscovery), "restAPI") {

		DSMasterIP = (string(responseFromDiscovery[0 : len(string(responseFromDiscovery))-7]))
		return
	}
}

func analyzeRequest(r *http.Request) string {
	requestBody, err := ioutil.ReadAll(r.Body) //Read the request
	if err != nil {
		fmt.Println("An error has occurred trying to read client's request. ")
		fmt.Println(err.Error())
		return ""
	}
	var receivedRequest string                                  //Put client's request in a string
	err = json.Unmarshal([]byte(requestBody), &receivedRequest) //Unmarshal client's request
	if err != nil {
		fmt.Println("Error unmarshaling client's request.")
		fmt.Println(err.Error())
		return ""
	}
	return receivedRequest //Return unmarshaled string
}
