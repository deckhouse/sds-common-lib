package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func main() {
	// Example JSON payload (in a real program this would come from e.g. an HTTP response body)
	jsonStr := `{"name":"Alice","age":30}`

	// Wrap the JSON string in an io.ReadCloser to simulate something like http.Response.Body
	rc := io.NopCloser(strings.NewReader(jsonStr))
	defer rc.Close()

	// Unmarshal the JSON from the io.ReadCloser into a Go struct
	var p Person
	if err := json.NewDecoder(rc).Decode(&p); err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", p)
}
