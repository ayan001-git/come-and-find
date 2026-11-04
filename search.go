package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type SearchIntent struct {
	Query    string   `json:"query"`
	Terms    []string `json:"terms"`
	Category string   `json:"category"`
	Min      *float64 `json:"min_price,omitempty"`
	Max      *float64 `json:"max_price,omitempty"`
}

var pricePattern = regexp.MustCompile(`(?i)(не дороже|не дешевле|up to|at least|under|below|above|from|до|от)\s+(\d+(?:[.,]\d+)?)\s*(тысяч[а-я]*|тыс\.?|thousand|k\b)?\s*(?:тенге|kzt|₸)?`)
var wordsPattern = regexp.MustCompile(`[\p{L}\p{N}]+`)
var categoryAliases = map[string]string{
	"мышь": "Mice", "мыши": "Mice", "мышку": "Mice", "мышки": "Mice", "мышка": "Mice", "mouse": "Mice", "mice": "Mice",
	"клавиатура": "Keyboards", "клавиатуру": "Keyboards", "клавиатуры": "Keyboards", "keyboard": "Keyboards", "keyboards": "Keyboards",
	"камера": "Webcams", "камеру": "Webcams", "веб": "Webcams", "вебкамера": "Webcams", "вебкамеру": "Webcams", "webcam": "Webcams", "webcams": "Webcams",
	"монитор": "Monitors", "мониторы": "Monitors", "монитора": "Monitors", "monitor": "Monitors", "monitors": "Monitors",
	"ssd": "Storage", "накопитель": "Storage", "накопители": "Storage", "storage": "Storage",
}
var queryAliases = map[string]string{"беспроводная": "wireless", "беспроводную": "wireless", "беспроводной": "wireless", "проводная": "usb", "проводную": "usb", "клавиш": "keys", "офисная": "office", "офисную": "office", "тихая": "quiet", "тихую": "quiet", "тихой": "quiet", "портативный": "portable", "портативного": "portable"}
var fillers = map[string]bool{"найди": true, "найти": true, "покажи": true, "мне": true, "нужна": true, "нужен": true, "пожалуйста": true, "купить": true, "товары": true, "товар": true, "find": true, "show": true, "me": true, "a": true, "an": true, "the": true, "please": true, "i": true, "want": true, "тенге": true, "kzt": true, "и": true, "and": true, "для": true, "for": true, "камеры": true}

// Deterministic catalogue rules, not a general-purpose language model.
func parseSearch(input string) (SearchIntent, error) {
	out := SearchIntent{Query: strings.TrimSpace(input), Terms: []string{}}
	if len([]rune(input)) > 300 {
		return out, fmt.Errorf("Search must be at most 300 characters")
	}
	text := strings.ToLower(strings.ReplaceAll(input, "ё", "е"))
	grouped := regexp.MustCompile(`(\d)[ \x{00A0}](\d{3})(?:\b)`)
	for grouped.MatchString(text) {
		text = grouped.ReplaceAllString(text, "${1}${2}")
	}
	numbers := map[string]string{"одной": "1", "одну": "1", "один": "1", "двух": "2", "две": "2", "два": "2", "трех": "3", "три": "3", "четырех": "4", "четыре": "4", "пяти": "5", "пять": "5", "шести": "6", "шесть": "6", "семи": "7", "семь": "7", "восьми": "8", "восемь": "8", "девяти": "9", "девять": "9", "десяти": "10", "десять": "10", "двадцати": "20", "двадцать": "20", "тридцати": "30", "тридцать": "30", "пятидесяти": "50", "пятьдесят": "50", "ста": "100", "сто": "100", "one": "1", "two": "2", "three": "3", "five": "5", "ten": "10", "twenty": "20", "fifty": "50", "hundred": "100"}
	text = wordsPattern.ReplaceAllStringFunc(text, func(w string) string {
		if n, ok := numbers[w]; ok {
			return n
		}
		return w
	})
	var parseErr error
	text = pricePattern.ReplaceAllStringFunc(text, func(chunk string) string {
		m := pricePattern.FindStringSubmatch(chunk)
		v, err := strconv.ParseFloat(strings.ReplaceAll(m[2], ",", "."), 64)
		if err != nil {
			parseErr = fmt.Errorf("Invalid search price")
			return ""
		}
		if m[3] != "" {
			v *= 1000
		}
		if v > 9999999999.99 {
			parseErr = fmt.Errorf("Search price is too large")
			return ""
		}
		lower := m[1] == "от" || m[1] == "from" || m[1] == "above" || m[1] == "at least" || m[1] == "не дешевле"
		if lower {
			if out.Min == nil || v > *out.Min {
				x := v
				out.Min = &x
			}
		} else {
			if out.Max == nil || v < *out.Max {
				x := v
				out.Max = &x
			}
		}
		return " "
	})
	if parseErr != nil {
		return out, parseErr
	}
	if out.Min != nil && out.Max != nil && *out.Min > *out.Max {
		return out, fmt.Errorf("Minimum price exceeds maximum")
	}
	seen := map[string]bool{}
	for _, word := range wordsPattern.FindAllString(text, -1) {
		if cat, ok := categoryAliases[word]; ok {
			if out.Category != "" && out.Category != cat {
				return out, fmt.Errorf("Search one product category at a time")
			}
			out.Category = cat
			continue
		}
		if fillers[word] {
			continue
		}
		if alias, ok := queryAliases[word]; ok {
			word = alias
		}
		if !seen[word] {
			out.Terms = append(out.Terms, word)
			seen[word] = true
		}
	}
	return out, nil
}
func (a *App) interpretSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	intent, err := parseSearch(r.URL.Query().Get("q"))
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, intent)
}
