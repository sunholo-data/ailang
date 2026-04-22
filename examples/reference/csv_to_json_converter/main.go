package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	csv := "name,age,email\nAlice,30,alice@example.com\nBob,25,bob@example.com\nCarol,35,carol@example.com"
	os.WriteFile("users.csv", []byte(csv), 0644)

	data, _ := os.ReadFile("users.csv")
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	headers := strings.Split(lines[0], ",")

	type Row map[string]interface{}
	var rows []Row

	for _, line := range lines[1:] {
		vals := strings.Split(line, ",")
		obj := Row{}
		for i, h := range headers {
			obj[h] = vals[i]
		}
		age, err := strconv.Atoi(fmt.Sprintf("%v", obj["age"]))
		if err != nil || age <= 0 {
			continue
		}
		if !strings.Contains(fmt.Sprintf("%v", obj["email"]), "@") {
			continue
		}
		obj["age"] = age
		rows = append(rows, obj)
	}

	out, _ := json.MarshalIndent(rows, "", "  ")
	os.WriteFile("users.json", out, 0644)
	fmt.Printf("Converted %d valid rows to users.json\n", len(rows))
}
