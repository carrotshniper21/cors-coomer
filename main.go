package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

// semaphore channel to control concurrency
var semaphore = make(chan bool, 1)

func main() {
	PORT := os.Getenv("PORT")
	if PORT == "" {
		PORT = "6969"
	}

	http.HandleFunc("/", cors_coomer)
	fmt.Println("Server is running on port " + PORT)
	fmt.Println("http://localhost:" + PORT + "/?url=https://example.com" + "&method=GET")
	fmt.Print("Press Ctrl + C to stop the server")

	// start the server
	http.ListenAndServe(":"+PORT, nil)
}

// cors_coomer function that will be called when the endpoint is hit
func cors_coomer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	// get the url and method from request params
	url := r.URL.Query().Get("url")
	method := r.URL.Query().Get("method")
	if method == "" {
		method = "GET"
	}

	// control concurrency by blocking the semaphore channel until a spot is available
	semaphore <- true
	defer func() {
		<-semaphore
	}()

	// create a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// read the request body if it's a POST request
	var requestBody io.Reader
	if r.Method == http.MethodPost {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		requestBody = bytes.NewReader(body)
	}

	// make a request to the url and get the response body
	req, err := http.NewRequestWithContext(ctx, method, url, requestBody)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36")
	if err != nil {
		// write error with fprintf
		fmt.Fprintf(w, "Some crazy ass shit is going on: %s", err.Error())
		return
	}

	// copy headers from the original request to the outgoing request
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// write error with fprintf
		fmt.Fprintf(w, "Some crazy ass shit is going on: %s", err.Error())
		return
	}

	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	// parse response body before write to response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	w.Write(body)
	if resp.StatusCode == 200 {
		fmt.Println("Success on " + url + " with status code " + resp.Status)
	} else {
		fmt.Fprintf(w, "Failed on %s with status code %s", url, resp.Status)
		// waiting time before making another request
		time.Sleep(3 * time.Second)

		// append to a file when failed
		go func() {
			f, err := os.OpenFile("failed.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Println(err)
			}
			defer f.Close()

			if _, err := f.WriteString(url + " " + resp.Status + " " + time.Now().String() +
				"\n" + string(body) + "\n\n"); err != nil {
				fmt.Println(err)
			}
		}()
	}
}
