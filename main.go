package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"net/http"
	"encoding/json"
)

// db functions

func dbInit() {
	if _, err := os.Stat("db.txt"); errors.Is(err, fs.ErrNotExist){
		// creates the db.txt file if it doesn't exist
		fmt.Println("Creating db")
		_, err := os.Create("db.txt") 
		if err != nil {
			panic(err)
		}	
	}
}

func getEntries() map[string]string {
	fileName := "db.txt"
	data, err := os.ReadFile(fileName)
	if err != nil {
		panic(err)
	}
	stringData := string(data)
	splitData := strings.Split(stringData, "\n")
	shortenedLinkToDestinationLink := make(map[string]string)
	for i := 0; i < len(splitData); i++ {
		item := splitData[i]
		item = strings.TrimSpace(item)
		if len(item) > 0{
			splitItem := strings.Split(item, "{%$delimiter$%}")
			shortened, destination := splitItem[0], splitItem[1]
			shortenedLinkToDestinationLink[shortened] = destination
		}
	}
	return shortenedLinkToDestinationLink
}

func addEntry(shortened string, destination string) error {
	// Open the file in "append" mode, so we don't overwrite what's already there.
	file, err := os.OpenFile("db.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	
	if err != nil {
		return err
	}

	defer file.Close()

	if _, err := file.WriteString(shortened + "{%$delimiter$%}" + destination + "\n"); err != nil {
		return err
	}

	return nil
}

func clearDb() {
	err := os.Truncate("db.txt", 0)
	if err != nil {
		panic(err)
	}
	fmt.Println("db cleared!")
}

func getDestinationFromShortened(shortened string) (string, error) {
	destination := getEntries()[shortened]
	destination = strings.TrimSpace(destination)
	if len(destination) > 0 {
		return destination, nil
	} else {
		return "", errors.New("shortened link does not exist in db")
	}
}

// api functions

type registerSiteRequest struct {
	ShortenedLink string `json:"shortened"`
	DestinationLink string `json:"destination"`
}

type registerSiteResponse struct {
	Success bool `json:"success"`
}

func registerSiteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed (Must be POST)", http.StatusMethodNotAllowed)
		return
	}

	var req registerSiteRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	entries := getEntries()

	for key := range entries {
		if key == req.ShortenedLink {
			fmt.Println("Shortened Link Already Exists (Didn't add it)", req)
			res := registerSiteResponse{Success: false}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(res)

			return
		}
	}

	if strings.TrimSpace(req.ShortenedLink) == "" || strings.TrimSpace(req.DestinationLink) == "" {
		fmt.Println("Empty Body (Didn't add it)", req)
		res := registerSiteResponse{Success: false}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)

		return
	} 

	// add data to db 
	fmt.Println("Adding entry", req)
	addEntryErr := addEntry(req.ShortenedLink, req.DestinationLink)

	var hadSuccess bool 

	if addEntryErr == nil {
		hadSuccess = true
	} else {
		hadSuccess = false
	}

	res := registerSiteResponse{Success: hadSuccess}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)

}

func makeShortenedUrlHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/makeShortening.html")
}

func main() {
	dbInit()
	// clearDb()

	fmt.Println("Server started!")

	http.HandleFunc("/registerSite", registerSiteHandler)
	http.HandleFunc("/makeShortenedUrl", makeShortenedUrlHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// handle site redirects 
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() == "/" {
			http.Redirect(w, r, "/makeShortenedUrl", http.StatusSeeOther)
		}
		redirectDestination, err := getDestinationFromShortened(r.URL.String())
		fmt.Println(redirectDestination, err, r.URL.String())
		if err != nil {
			http.ServeFile(w, r, "templates/404.html")
		} else {
			http.Redirect(w, r, redirectDestination, http.StatusSeeOther)
		}
	})

	panic(http.ListenAndServe(":8000", nil))
}
