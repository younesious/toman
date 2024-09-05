package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

const (
	MinNumber = 0
	MaxNumber = 9999999999
)

var (
	units    = []string{"", "یک", "دو", "سه", "چهار", "پنج", "شش", "هفت", "هشت", "نه"}
	tens     = []string{"", "", "بیست", "سی", "چهل", "پنجاه", "شصت", "هفتاد", "هشتاد", "نود"}
	teens    = []string{"ده", "یازده", "دوازده", "سیزده", "چهارده", "پانزده", "شانزده", "هفده", "هجده", "نوزده"}
	hundreds = []string{"", "صد", "دویست", "سیصد", "چهارصد", "پانصد", "ششصد", "هفتصد", "هشتصد", "نهصد"}
	scales   = []string{"", "هزار", "میلیون"}
)

func convertToToman(number int64) int64 {
	return number / 10
}

func convertThreeDigits(n int) string {
	if n == 0 {
		return ""
	}

	result := make([]string, 0, 3)

	h := n / 100
	t := (n % 100) / 10
	u := n % 10

	if h > 0 {
		result = append(result, hundreds[h])
	}
	if t == 1 {
		result = append(result, teens[u])
	} else {
		if t > 1 {
			result = append(result, tens[t])
		}
		if u > 0 {
			result = append(result, units[u])
		}
	}

	return strings.Join(result, " و ")
}

func convertSegment(segment int, scale int) string {
	part := convertThreeDigits(segment)
	if scale > 0 && part != "" {
		part += " " + scales[scale]
	}
	return part
}

func convertToPersian(number int64) string {
	if number == 0 {
		return "صفر"
	}

	var wg sync.WaitGroup
	results := make([]string, 3) // Slice to store results in order
	var mu sync.Mutex            // Mutex for safe concurrent writes and fix race condition

	for scale := 0; scale < 3 && number > 0; scale++ {
		segment := int(number % 1000)
		wg.Add(1)
		go func(segment, scale int) {
			defer wg.Done()
			part := convertSegment(segment, scale)
			if part != "" {
				mu.Lock()
				results[scale] = part // Store result in the correct order
				mu.Unlock()
			}
		}(segment, scale)
		number /= 1000
	}

	wg.Wait()

	var parts []string
	for i := len(results) - 1; i >= 0; i-- {
		if results[i] != "" {
			parts = append(parts, results[i])
		}
	}

	return strings.Join(parts, " و ")
}

func handleConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		Number string `json:"number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON input", http.StatusBadRequest)
		return
	}

	number, err := strconv.ParseInt(input.Number, 10, 64)
	if err != nil || number < MinNumber || number > MaxNumber {
		http.Error(w, "Invalid number (must be between 0 and 9999999999)", http.StatusBadRequest)
		return
	}

	tomanNumber := convertToToman(number)
	persianWords := convertToPersian(tomanNumber)

	response := struct {
		Words string `json:"words"`
	}{
		Words: persianWords + " تومان",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
	}
}

func main() {
	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/convert", handleConvert)

	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
