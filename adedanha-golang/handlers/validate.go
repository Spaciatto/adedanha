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

const (
	StatusValid     = "valid"
	StatusInvalid   = "invalid"
	StatusUncertain = "uncertain"
)

type FieldValidation struct {
	Field  string `json:"field"`
	Value  string `json:"value"`
	Status string `json:"status"`
}

type PlayerValidation struct {
	UserID         string            `json:"user_id"`
	Validations    []FieldValidation `json:"validations"`
	SuggestedScore int               `json:"suggested_score"`
}

var httpClient = &http.Client{
	Timeout: 8 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: false,
	},
}

const userAgent = "AdedanhaOnlineBot/1.0 (https://github.com/Spaciatto/adedanha; adedanha-game) Go-http-client"

// --- Local lists for precise validation ---

var knownColors = map[string]bool{
	"AMARELO": true, "AZUL": true, "BRANCO": true, "BEGE": true, "BORDO": true,
	"BRONZE": true, "CARAMELO": true, "CARMESIM": true, "CASTANHO": true,
	"CINZA": true, "CORAL": true, "CREME": true, "DOURADO": true,
	"ESCARLATE": true, "FUCSIA": true, "GRAFITE": true, "GRENA": true,
	"INDIGO": true, "LARANJA": true, "LAVANDA": true, "LILAS": true,
	"MAGENTA": true, "MARFIM": true, "MARROM": true, "MOSTARDA": true,
	"NEGRO": true, "OCRE": true, "OLIVA": true, "OURO": true,
	"PRATA": true, "PRETO": true, "PURPURA": true, "ROSA": true,
	"ROXO": true, "RUBRO": true, "SALMAO": true, "SEPIA": true,
	"TERRACOTA": true, "TURQUESA": true, "VERDE": true, "VERMELHO": true,
	"VINHO": true, "VIOLETA": true, "CIANO": true, "CELESTE": true,
	"FERRUGEM": true, "NUDE": true, "PETROLEO": true, "TIJOLO": true,
}

var knownFruits = map[string]bool{
	"ABACATE": true, "ABACAXI": true, "ABIU": true, "ABRICÓ": true, "ABRICO": true,
	"ACAI": true, "AÇAI": true, "ACEROLA": true, "AKEE": true, "AMEIXA": true,
	"AMORA": true, "ANANÁS": true, "ANANAS": true, "ARACA": true, "ARAÇA": true,
	"ATEMOIA": true, "AVELA": true, "AVELÃ": true, "BACABA": true, "BACURI": true,
	"BANANA": true, "BERGAMOTA": true, "BIRIBA": true, "BURITI": true,
	"BUTIA": true, "BUTIÁ": true, "CABELUDINHA": true, "CACAU": true,
	"CAGAITA": true, "CAIMITO": true, "CAJA": true, "CAJÁ": true,
	"CAJAMANGA": true, "CAJU": true, "CALAMANSI": true, "CAMBUCA": true,
	"CAMBUCI": true, "CAMU-CAMU": true, "CAQUI": true, "CARAMBOLA": true,
	"CASTANHA": true, "CEREJA": true, "CIRIGUELA": true, "COCO": true,
	"CUPUACU": true, "CUPUAÇU": true, "DAMASCO": true, "DEKOPON": true,
	"DURIÃO": true, "DURIAO": true, "EMBAUBA": true, "FEIJOA": true,
	"FIGO": true, "FRAMBOESA": true, "FRUTA-PAO": true, "GABIROBA": true,
	"GOIABA": true, "GRANADILHA": true, "GRAVIOLA": true, "GROSELHA": true,
	"GRUMIXAMA": true, "GUABIROBA": true, "GUARANA": true, "GUARANÁ": true,
	"ILAMA": true, "IMBE": true, "INGA": true, "INGÁ": true,
	"JABUTICABA": true, "JACA": true, "JAMBO": true, "JAMBOLAO": true,
	"JAMELAO": true, "JATOBA": true, "JATOBÁ": true, "JENIPAPO": true,
	"JUÇARA": true, "JUCARA": true, "KIWI": true, "KUMQUAT": true,
	"LARANJA": true, "LICHIA": true, "LIMA": true, "LIMAO": true, "LIMÃO": true,
	"LONGAN": true, "LOQUAT": true, "LUCUMA": true, "MACA": true, "MAÇÃ": true,
	"MACADAMIA": true, "MAMAO": true, "MAMÃO": true, "MANGA": true,
	"MANGABA": true, "MANGOSTAO": true, "MANGOSTIM": true, "MARACUJA": true,
	"MARACUJÁ": true, "MARMELO": true, "MELANCIA": true, "MELAO": true,
	"MELÃO": true, "MEXERICA": true, "MIRTILO": true, "MORANGO": true,
	"MURICI": true, "NECTARINA": true, "NONI": true, "NOZ": true,
	"PERA": true, "PEQUI": true, "PESSEGO": true, "PÊSSEGO": true,
	"PHYSALIS": true, "PINHA": true, "PITANGA": true, "PITAIA": true,
	"PITAYA": true, "POMELO": true, "PUPUNHA": true, "RAMBUTAN": true,
	"ROMA": true, "ROMÃ": true, "SAPOTI": true, "SAPUCAIA": true,
	"SERIGUELA": true, "SIRIGUELA": true, "TAMARINDO": true, "TAMARA": true,
	"TANGERINA": true, "TUCUMA": true, "TUCUMÃ": true, "TUNA": true,
	"UMBU": true, "UVA": true, "UVAIA": true,
}

var knownAnimals = map[string]bool{
	"ABELHA": true, "ABUTRE": true, "AGUIA": true, "ÁGUIA": true,
	"ALBATROZ": true, "ALCE": true, "ALPACA": true, "ANACONDA": true,
	"ANTA": true, "ANTILOPE": true, "ARANHA": true, "ARARA": true,
	"ARARINHA": true, "ARMADILHO": true, "ATUM": true, "AVESTRUZ": true,
	"BABUINO": true, "BACALHAU": true, "BALEIA": true, "BARATAS": true,
	"BARATA": true, "BEIJA-FLOR": true, "BESOURO": true, "BISAO": true,
	"BISONTE": true, "BOA": true, "BODE": true, "BOI": true, "BORBOLETA": true,
	"BOTO": true, "BÚFALO": true, "BUFALO": true, "BURRO": true,
	"CABRA": true, "CACATUA": true, "CACHORRO": true, "CAGADO": true,
	"CAIMAO": true, "CALANGO": true, "CAMALEAO": true, "CAMELO": true,
	"CAMUNDONGO": true, "CANARIO": true, "CANGURU": true, "CAPIVARA": true,
	"CARACOL": true, "CARANGUEJO": true, "CARDEAL": true, "CARNEIRO": true,
	"CASTOR": true, "CAVALO": true, "CERVO": true, "CHIMPANZE": true,
	"CHINCHILA": true, "CIGARRA": true, "CISNE": true, "COALA": true,
	"COBRA": true, "CODORNA": true, "COELHO": true, "COLIBRI": true,
	"CONDOR": true, "CORUJA": true, "CORVO": true, "COTIA": true,
	"CROCODILO": true, "CUPIM": true, "CUTIA": true,
	"DELFIM": true, "DINOSSAURO": true, "DONINHA": true, "DROMEDARIO": true,
	"ELEFANTE": true, "EMU": true, "ENGUIA": true, "ESCORPIAO": true,
	"ESQUILO": true, "ESTRELA-DO-MAR": true,
	"FALCAO": true, "FLAMINGO": true, "FOCA": true, "FORMIGA": true,
	"FURÃO": true, "FURAO": true,
	"GAFANHOTO": true, "GAIVOTA": true, "GALINHA": true, "GALO": true,
	"GAMBÁ": true, "GAMBA": true, "GANSO": true, "GARÇA": true, "GARCA": true,
	"GATO": true, "GAVIAO": true, "GAZELA": true, "GIRAFA": true,
	"GOLFINHO": true, "GORILA": true, "GRALHA": true, "GRILO": true,
	"GUAXINIM": true, "GUEPARDO": true,
	"HAMSTER": true, "HIENA": true, "HIPOPOTAMO": true, "HIPOPÓTAMO": true,
	"IGUANA": true, "IMPALA": true,
	"JABUTI": true, "JACARE": true, "JACARÉ": true, "JAGUAR": true,
	"JAGUATIRICA": true, "JAVALI": true, "JIBOIA": true, "JOANINHA": true,
	"LAGARTA": true, "LAGARTIXA": true, "LAGARTO": true, "LAGOSTA": true,
	"LEAO": true, "LEÃO": true, "LEBRE": true, "LEMURE": true,
	"LEOPARDO": true, "LESMA": true, "LIBÉLULA": true, "LIBELULA": true,
	"LINCE": true, "LOBO": true, "LONTRA": true, "LULA": true,
	"MACACO": true, "MAMBA": true, "MANTA": true, "MORCEGO": true,
	"MORSA": true, "MOSCA": true, "MOSQUITO": true, "MULA": true,
	"NARVAL": true, "NAJA": true,
	"OCELOTE": true, "ONÇA": true, "ONCA": true, "ORANGOTANGO": true,
	"ORCA": true, "ORNITORRINCO": true, "OSTRA": true, "OVELHA": true,
	"PANDA": true, "PANTERA": true, "PAPAGAIO": true, "PATO": true,
	"PAVAO": true, "PAVÃO": true, "PEIXE": true, "PELICANO": true,
	"PERDIZ": true, "PERIQUITO": true, "PERU": true, "PICA-PAU": true,
	"PINGUIM": true, "PIRANHA": true, "POLVO": true, "POMBO": true,
	"PORCO": true, "PREGUIÇA": true, "PREGUICA": true, "PUMA": true,
	"QUATI":  true,
	"RAPOSA": true, "RATO": true, "RAIA": true, "RINOCERONTE": true,
	"ROUXINOL": true,
	"SABIA":    true, "SABIÁ": true, "SALAMANDRA": true, "SALMAO": true,
	"SAPO": true, "SARDINHA": true, "SERPENTE": true, "SURICATO": true,
	"TAMANDUÁ": true, "TAMANDUA": true, "TAPIR": true, "TARTARUGA": true,
	"TATU": true, "TIGRE": true, "TILAPIA": true, "TUCANO": true,
	"TUBARAO": true, "TUBARÃO": true, "TOUPEIRA": true,
	"URUBU": true, "URSO": true,
	"VACA": true, "VEADO": true, "VESPA": true, "VIUVA-NEGRA": true,
	"ZEBRA": true,
}

// citiesCache stores Brazilian cities loaded from IBGE on first use
var citiesCache map[string]bool
var citiesCacheOnce sync.Once

func loadCitiesCache() {
	citiesCache = make(map[string]bool)
	log.Println("[VALIDATE] Carregando lista de cidades do IBGE...")

	req, err := http.NewRequest("GET", "https://servicodados.ibge.gov.br/api/v1/localidades/municipios", nil)
	if err != nil {
		log.Printf("[VALIDATE] Erro ao criar request IBGE cidades: %v", err)
		return
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("[VALIDATE] Erro ao carregar cidades IBGE: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[VALIDATE] Erro ao ler resposta IBGE cidades: %v", err)
		return
	}

	var cities []struct {
		Nome string `json:"nome"`
	}
	if err := json.Unmarshal(body, &cities); err != nil {
		log.Printf("[VALIDATE] Erro ao parsear cidades IBGE: %v", err)
		return
	}

	for _, c := range cities {
		normalized := strings.ToUpper(removeAccents(c.Nome))
		citiesCache[normalized] = true
		// Also store with accents
		citiesCache[strings.ToUpper(c.Nome)] = true
	}
	log.Printf("[VALIDATE] %d cidades carregadas do IBGE", len(cities))
}

// --- Wikidata SPARQL for semantic validation ---

// Wikidata category IDs
const (
	wdFilm   = "Q11424"  // film
	wdObject = "Q223557" // physical object (broad)
)

func wikidataCheck(label string, categoryQID string) string {
	// SPARQL ASK query: is there an item with this Portuguese label that is instance of category$1
	sparql := fmt.Sprintf(`ASK {
		$2item rdfs:label "%s"@pt .
		$1item wdt:P31/wdt:P279* wd:%s .
	}`, strings.ToLower(label), categoryQID)

	apiURL := fmt.Sprintf("https://query.wikidata.org/sparql?query=%s&format=json", url.QueryEscape(sparql))

	log.Printf("[VALIDATE] Wikidata SPARQL: '%s' (categoria: %s) — query enviada", label, categoryQID)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Printf("[VALIDATE] Wikidata: erro ao criar request: %v", err)
		return StatusUncertain
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/sparql-results+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("[VALIDATE] Wikidata: erro na consulta para '%s': %v", label, err)
		return StatusUncertain
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[VALIDATE] Wikidata: erro ao ler resposta para '%s': %v", label, err)
		return StatusUncertain
	}

	log.Printf("[VALIDATE] Wikidata: '%s' — resposta (status %d): %s", label, resp.StatusCode, truncateLog(string(body), 200))

	var result struct {
		Boolean bool `json:"boolean"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		// Try alternate format — sometimes Wikidata returns differently
		log.Printf("[VALIDATE] Wikidata: erro ao parsear JSON para '%s': %v", label, err)
		return StatusUncertain
	}

	if result.Boolean {
		log.Printf("[VALIDATE] Wikidata: '%s' (categoria: %s) → VÁLIDO", label, categoryQID)
		return StatusValid
	}
	log.Printf("[VALIDATE] Wikidata: '%s' (categoria: %s) → INVÁLIDO", label, categoryQID)
	return StatusInvalid
}

// --- IBGE Names API ---

func validateNameIBGE(value string) string {
	nameClean := strings.TrimSpace(value)
	if nameClean == "" {
		return StatusInvalid
	}

	apiURL := fmt.Sprintf("https://servicodados.ibge.gov.br/api/v2/censos/nomes/%s", url.PathEscape(nameClean))
	log.Printf("[VALIDATE] Nome IBGE: consultando '%s' — URL: %s", nameClean, apiURL)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return StatusUncertain
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("[VALIDATE] Nome IBGE: erro para '%s': %v", nameClean, err)
		return StatusUncertain
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return StatusUncertain
	}

	log.Printf("[VALIDATE] Nome IBGE: '%s' — resposta (status %d): %s", nameClean, resp.StatusCode, truncateLog(string(body), 150))

	var result []interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return StatusUncertain
	}

	if len(result) > 0 {
		log.Printf("[VALIDATE] Nome IBGE: '%s' → VÁLIDO", nameClean)
		return StatusValid
	}
	log.Printf("[VALIDATE] Nome IBGE: '%s' → INCERTO (não encontrado)", nameClean)
	return StatusUncertain
}

// --- Main validation handler ---

func ValidateRound(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID := vars["id"]
	roundID := vars["roundId"]

	var letter string
	if err := database.DB.QueryRow(
		"SELECT letter FROM rounds WHERE id = $1 AND match_id = $2", roundID, matchID,
	).Scan(&letter); err != nil {
		http.Error(w, `{"error":"Round not found"}`, http.StatusNotFound)
		return
	}

	rows, err := database.DB.Query(
		"SELECT user_id, color, fruit, object, movie, city, animal, name FROM answers WHERE round_id = $1",
		roundID,
	)
	if err != nil {
		http.Error(w, `{"error":"Failed to get answers"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type answerRow struct {
		UserID, Color, Fruit, Object, Movie, City, Animal, Name string
	}

	var answers []answerRow
	for rows.Next() {
		var a answerRow
		if err := rows.Scan(&a.UserID, &a.Color, &a.Fruit, &a.Object, &a.Movie, &a.City, &a.Animal, &a.Name); err == nil {
			answers = append(answers, a)
		}
	}

	// Ensure cities cache is loaded
	citiesCacheOnce.Do(loadCitiesCache)

	log.Printf("[VALIDATE] Iniciando validação — rodada %s, letra %s, %d jogadores", roundID, letter, len(answers))

	var wg sync.WaitGroup
	results := make([]PlayerValidation, len(answers))

	for i, a := range answers {
		wg.Add(1)
		go func(idx int, ans answerRow) {
			defer wg.Done()
			pv := PlayerValidation{UserID: ans.UserID}

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

			log.Printf("[VALIDATE] Jogador %s: cor='%s' fruta='%s' objeto='%s' filme='%s' cidade='%s' animal='%s' nome='%s'",
				ans.UserID, ans.Color, ans.Fruit, ans.Object, ans.Movie, ans.City, ans.Animal, ans.Name)

			score := 0
			for _, f := range fields {
				var status string
				if f.value == "" {
					status = StatusInvalid
				} else if !startsWithLetter(f.value, letter) {
					log.Printf("[VALIDATE] %s: '%s' → INVÁLIDO (não começa com '%s')", f.name, f.value, letter)
					status = StatusInvalid
				} else {
					status = validateField(f.name, f.value)
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
	log.Printf("[VALIDATE] Validação concluída — rodada %s", roundID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// --- Field validation dispatcher ---

func validateField(field, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return StatusInvalid
	}

	switch field {
	case "color":
		return validateFromList(value, knownColors, "Cor")
	case "fruit":
		return validateFromList(value, knownFruits, "Fruta")
	case "animal":
		return validateFromList(value, knownAnimals, "Animal")
	case "city":
		return validateCity(value)
	case "name":
		return validateNameIBGE(value)
	case "movie":
		return wikidataCheck(value, wdFilm)
	case "object":
		return wikidataCheck(value, wdObject)
	default:
		return StatusUncertain
	}
}

// --- Helpers ---

func validateFromList(value string, list map[string]bool, label string) string {
	normalized := strings.ToUpper(removeAccents(value))
	if list[normalized] || list[strings.ToUpper(value)] {
		log.Printf("[VALIDATE] %s: '%s' → VÁLIDO (lista local)", label, value)
		return StatusValid
	}
	log.Printf("[VALIDATE] %s: '%s' → INVÁLIDO (não encontrado na lista)", label, value)
	return StatusInvalid
}

func validateCity(value string) string {
	normalized := strings.ToUpper(removeAccents(value))
	if citiesCache[normalized] || citiesCache[strings.ToUpper(value)] {
		log.Printf("[VALIDATE] Cidade: '%s' → VÁLIDO (IBGE cache)", value)
		return StatusValid
	}
	log.Printf("[VALIDATE] Cidade: '%s' → INVÁLIDO (não encontrado no IBGE)", value)
	return StatusInvalid
}

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

func truncateLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
