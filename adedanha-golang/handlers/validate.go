package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode"

	"adedanha-golang/database"

	"github.com/gorilla/mux"
)

// Validation status constants
const (
	StatusValid     = "valid"
	StatusInvalid   = "invalid"
	StatusUncertain = "uncertain"
)

// FieldValidation represents the validation result for a single field
type FieldValidation struct {
	Field  string `json:"field"`
	Value  string `json:"value"`
	Status string `json:"status"` // valid, invalid, uncertain
}

// PlayerValidation represents all field validations for a player
type PlayerValidation struct {
	UserID         string            `json:"user_id"`
	Validations    []FieldValidation `json:"validations"`
	SuggestedScore int               `json:"suggested_score"`
}

// Known colors in Portuguese
var knownColors = map[string]bool{
	"AMARELO": true, "AZUL": true, "BRANCO": true, "BEGE": true, "BORDO": true,
	"BRONZE": true, "CARAMELO": true, "CARMESIM": true, "CASTANHO": true,
	"CINZA": true, "CORAL": true, "CREME": true, "DOURADO": true,
	"ESCARLATE": true, "FÚCSIA": true, "FUCSIA": true, "GRAFITE": true,
	"GRENÁ": true, "GRENA": true, "ÍNDIGO": true, "INDIGO": true,
	"LARANJA": true, "LAVANDA": true, "LILÁS": true, "LILAS": true,
	"MAGENTA": true, "MARFIM": true, "MARROM": true, "MOSTARDA": true,
	"NEGRO": true, "OCRE": true, "OLIVA": true, "OURO": true,
	"PRATA": true, "PRETO": true, "PÚRPURA": true, "PURPURA": true,
	"ROSA": true, "ROXO": true, "RUBRO": true, "SALMÃO": true, "SALMAO": true,
	"SÉPIA": true, "SEPIA": true, "TERRACOTA": true, "TURQUESA": true,
	"VERDE": true, "VERMELHO": true, "VINHO": true, "VIOLETA": true,
}

// HTTP client with timeout for external API calls
var httpClient = &http.Client{
	Timeout: 5 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: false,
	},
}

const userAgent = "AdedanhaOnlineBot/1.0 (https://github.com/Spaciatto/adedanha; adedanha-game) Go-http-client"

// wikiGet performs a GET request with proper User-Agent for Wikimedia APIs
func wikiGet(apiURL string) (*http.Response, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	return httpClient.Do(req)
}

// ValidateRound validates all answers for a round
func ValidateRound(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID := vars["id"]
	roundID := vars["roundId"]

	// Get round letter
	var letter string
	if err := database.DB.QueryRow(
		"SELECT letter FROM rounds WHERE id = ? AND match_id = ?", roundID, matchID,
	).Scan(&letter); err != nil {
		http.Error(w, `{"error":"Round not found"}`, http.StatusNotFound)
		return
	}

	// Get all answers
	rows, err := database.DB.Query(
		"SELECT user_id, color, fruit, object, movie, city, animal, name FROM answers WHERE round_id = ?",
		roundID,
	)
	if err != nil {
		http.Error(w, `{"error":"Failed to get answers"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type answerRow struct {
		UserID string
		Color  string
		Fruit  string
		Object string
		Movie  string
		City   string
		Animal string
		Name   string
	}

	var answers []answerRow
	for rows.Next() {
		var a answerRow
		if err := rows.Scan(&a.UserID, &a.Color, &a.Fruit, &a.Object, &a.Movie, &a.City, &a.Animal, &a.Name); err == nil {
			answers = append(answers, a)
		}
	}

	// Validate all answers in parallel
	var wg sync.WaitGroup
	results := make([]PlayerValidation, len(answers))

	log.Printf("[VALIDATE] Iniciando validação da rodada %s (letra: %s) — %d jogadores", roundID, letter, len(answers))

	for i, a := range answers {
		wg.Add(1)
		go func(idx int, ans answerRow) {
			defer wg.Done()
			pv := PlayerValidation{UserID: ans.UserID}

			log.Printf("[VALIDATE] Validando jogador %s: cor='%s' fruta='%s' objeto='%s' filme='%s' cidade='%s' animal='%s' nome='%s'",
				ans.UserID, ans.Color, ans.Fruit, ans.Object, ans.Movie, ans.City, ans.Animal, ans.Name)

			fields := []struct {
				name  string
				value string
			}{
				{"color", ans.Color},
				{"fruit", ans.Fruit},
				{"object", ans.Object},
				{"movie", ans.Movie},
				{"city", ans.City},
				{"animal", ans.Animal},
				{"name", ans.Name},
			}

			score := 0
			for _, f := range fields {
				var status string
				if f.value == "" {
					status = StatusInvalid
				} else if !startsWithLetter(f.value, letter) {
					status = StatusInvalid
				} else {
					status = validateField(f.name, f.value, letter)
				}

				pv.Validations = append(pv.Validations, FieldValidation{
					Field:  f.name,
					Value:  f.value,
					Status: status,
				})

				switch status {
				case StatusValid:
					score += 10
				case StatusUncertain:
					score += 5
				}
			}

			pv.SuggestedScore = score
			results[idx] = pv
			log.Printf("[VALIDATE] Jogador %s — score sugerido: %d", ans.UserID, score)
		}(i, a)
	}

	wg.Wait()

	log.Printf("[VALIDATE] Validação da rodada %s concluída", roundID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// startsWithLetter checks if a word starts with the given letter
func startsWithLetter(word, letter string) bool {
	if word == "" || letter == "" {
		return false
	}
	wordRunes := []rune(strings.ToUpper(strings.TrimSpace(word)))
	letterRunes := []rune(strings.ToUpper(letter))
	if len(wordRunes) == 0 || len(letterRunes) == 0 {
		return false
	}
	return wordRunes[0] == letterRunes[0]
}

// validateField validates a single field value
func validateField(field, value, letter string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return StatusInvalid
	}

	switch field {
	case "color":
		return validateColor(value)
	case "fruit":
		return validateWithWikipedia(value, "fruta")
	case "object":
		return validateWithWikipedia(value, "")
	case "movie":
		return validateWithWikipedia(value, "filme")
	case "city":
		return validateCity(value)
	case "animal":
		return validateWithWikipedia(value, "animal")
	case "name":
		return validateName(value)
	default:
		return StatusUncertain
	}
}

// validateColor checks against known color list
func validateColor(value string) string {
	normalized := strings.ToUpper(removeAccents(value))
	if knownColors[normalized] {
		log.Printf("[VALIDATE] Cor: '%s' → VÁLIDO (lista local)", value)
		return StatusValid
	}
	if knownColors[strings.ToUpper(value)] {
		log.Printf("[VALIDATE] Cor: '%s' → VÁLIDO (lista local, com acento)", value)
		return StatusValid
	}
	log.Printf("[VALIDATE] Cor: '%s' → INVÁLIDO (não encontrado na lista de cores)", value)
	return StatusInvalid
}

// validateCity checks using Wikipedia for cities
func validateCity(value string) string {
	return validateWithWikipedia(value, "cidade município")
}

// validateName checks using IBGE names API
func validateName(value string) string {
	nameClean := strings.TrimSpace(value)
	if nameClean == "" {
		return StatusInvalid
	}

	apiURL := fmt.Sprintf("https://servicodados.ibge.gov.br/api/v2/censos/nomes/%s", url.PathEscape(nameClean))
	log.Printf("[VALIDATE] Nome: consultando IBGE API — URL: %s", apiURL)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Printf("[VALIDATE] Nome: erro ao criar request para '%s': %v", nameClean, err)
		return StatusUncertain
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("[VALIDATE] Nome: erro na consulta IBGE para '%s': %v", nameClean, err)
		return StatusUncertain
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[VALIDATE] Nome: erro ao ler resposta IBGE para '%s': %v", nameClean, err)
		return StatusUncertain
	}

	log.Printf("[VALIDATE] Nome: '%s' — resposta IBGE (status %d): %s", nameClean, resp.StatusCode, truncateLog(string(body), 200))

	var result []interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("[VALIDATE] Nome: erro ao parsear JSON para '%s': %v", nameClean, err)
		return StatusUncertain
	}

	if len(result) > 0 {
		log.Printf("[VALIDATE] Nome: '%s' → VÁLIDO (encontrado no IBGE)", nameClean)
		return StatusValid
	}
	log.Printf("[VALIDATE] Nome: '%s' → INCERTO (não encontrado no IBGE)", nameClean)
	return StatusUncertain
}

// validateWithWikipedia searches Portuguese Wikipedia for the term
func validateWithWikipedia(value, category string) string {
	searchTerm := value
	if category != "" {
		searchTerm = value + " " + category
	}

	apiURL := fmt.Sprintf(
		"https://pt.wikipedia.org/w/api.php?action=query&list=search&srsearch=%s&srlimit=3&format=json",
		url.QueryEscape(searchTerm),
	)

	log.Printf("[VALIDATE] Wikipedia: consultando '%s' (categoria: '%s') — URL: %s", value, category, apiURL)

	resp, err := wikiGet(apiURL)
	if err != nil {
		log.Printf("[VALIDATE] Wikipedia: erro na consulta para '%s': %v", value, err)
		return StatusUncertain
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[VALIDATE] Wikipedia: erro ao ler resposta para '%s': %v", value, err)
		return StatusUncertain
	}

	log.Printf("[VALIDATE] Wikipedia: '%s' — resposta (status %d): %s", value, resp.StatusCode, truncateLog(string(body), 300))

	var wikiResp struct {
		Query struct {
			Search []struct {
				Title   string `json:"title"`
				Snippet string `json:"snippet"`
			} `json:"search"`
		} `json:"query"`
	}

	if err := json.Unmarshal(body, &wikiResp); err != nil {
		log.Printf("[VALIDATE] Wikipedia: erro ao parsear JSON para '%s': %v", value, err)
		return StatusUncertain
	}

	if len(wikiResp.Query.Search) == 0 {
		log.Printf("[VALIDATE] Wikipedia: '%s' (categoria: '%s') → INVÁLIDO (nenhum resultado)", value, category)
		return StatusInvalid
	}

	// Check if any result title closely matches the value
	valueUpper := strings.ToUpper(value)
	for _, result := range wikiResp.Query.Search {
		titleUpper := strings.ToUpper(result.Title)
		if titleUpper == valueUpper || strings.Contains(titleUpper, valueUpper) || strings.Contains(valueUpper, titleUpper) {
			log.Printf("[VALIDATE] Wikipedia: '%s' (categoria: '%s') → VÁLIDO (match: '%s')", value, category, result.Title)
			return StatusValid
		}
	}

	// Results found but no exact match
	titles := make([]string, 0, len(wikiResp.Query.Search))
	for _, r := range wikiResp.Query.Search {
		titles = append(titles, r.Title)
	}
	log.Printf("[VALIDATE] Wikipedia: '%s' (categoria: '%s') → INCERTO (resultados: %v)", value, category, titles)
	return StatusUncertain
}

// truncateLog truncates a string for logging purposes
func truncateLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// removeAccents removes common Portuguese accents for comparison
func removeAccents(s string) string {
	var result []rune
	for _, r := range s {
		switch {
		case r == 'Á' || r == 'À' || r == 'Ã' || r == 'Â' || r == 'á' || r == 'à' || r == 'ã' || r == 'â':
			result = append(result, 'A')
		case r == 'É' || r == 'Ê' || r == 'é' || r == 'ê':
			result = append(result, 'E')
		case r == 'Í' || r == 'í':
			result = append(result, 'I')
		case r == 'Ó' || r == 'Ô' || r == 'Õ' || r == 'ó' || r == 'ô' || r == 'õ':
			result = append(result, 'O')
		case r == 'Ú' || r == 'ú' || r == 'Ü' || r == 'ü':
			result = append(result, 'U')
		case r == 'Ç' || r == 'ç':
			result = append(result, 'C')
		case unicode.IsLetter(r):
			result = append(result, unicode.ToUpper(r))
		default:
			result = append(result, r)
		}
	}
	return string(result)
}
