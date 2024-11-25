package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const DATABASE_LOCAL_FILE = "./database.db"
const PORT_SRV = ":8080"
const URL_API = "https://economia.awesomeapi.com.br/json/last/USD-BRL"

type PriceResponse struct {
	USDBRL struct {
		Buy string `json:"bid"`
	} `json:"USD"`
}

func main() {
	//Cria o banco de dados
	db, err := sql.Open("sqlite3", DATABASE_LOCAL_FILE)
	if err != nil {
		log.Fatalf("falha ao tentar abrir o arquivo de banco de dados: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS price_quotation (id INTEGER PRIMARY KEY AUTOINCREMENT, value REAL, dateValue TIMESTAMP DEFAULT CURRENT_TIMESTAMP)")
	if err != nil {
		log.Fatalf("Erro ao criar a tabela: %v", err)
	}

	http.HandleFunc("/cotacao", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		priceQuotation, err := getCurrencyPrice(ctx)
		if err != nil {
			http.Error(w, "falha ao tentar obter cotação", http.StatusInternalServerError)
			log.Printf("erro no [GET] - cotação: %v", err)
			return
		}

		//inserir dados no database
		if err := insertDatabaseValues(ctx, db, priceQuotation); err != nil {
			http.Error(w, "falha ao tentar salvar cotação no banco de dados", http.StatusInternalServerError)
			log.Printf("erro ao gravar a cotação no banco de dados: %v", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"bid": priceQuotation.USDBRL.Buy})
	})

	log.Println("server started " + PORT_SRV + "...")
	log.Fatal(http.ListenAndServe(PORT_SRV, nil))
}

func getCurrencyPrice(ctx context.Context) (*PriceResponse, error) {
	// definir o client-HTTP com timeout
	client := &http.Client{Timeout: 200 * time.Millisecond}

	req, err := http.NewRequestWithContext(ctx, "GET", URL_API, nil)
	if err != nil {
		return nil, fmt.Errorf("ocorreu algum problema ao criar o request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ocorreu algum problema ao tentar realizar o request: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("response status: %s", resp.Status)

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("erro ao realizar o parse da resposta: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erro na resposta da API: %s", resp.Status)
	}

	var priceQuotation PriceResponse
	if usdrbl, ok := body["USDBRL"].(map[string]interface{}); ok {
		if bid, ok := usdrbl["bid"].(string); ok {
			priceQuotation.USDBRL.Buy = bid
		}
	}

	return &priceQuotation, nil
}

func insertDatabaseValues(ctx context.Context, db *sql.DB, priceQuotation *PriceResponse) error {
	ctxPersist, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()

	stmt, err := db.PrepareContext(ctxPersist, "INSERT INTO price_quotation (value) VALUES (?)")
	if err != nil {
		return fmt.Errorf("erro ao preparar insert: %w", err)
	}
	defer stmt.Close()

	convertPrice, err := strconv.ParseFloat(priceQuotation.USDBRL.Buy, 64)
	if err != nil {
		return fmt.Errorf("ocorreu algum erro ao tentar converter a cotação para float: %w", err)
	}

	//_, err = stmt.ExecContext(ctxPersist, priceQuotation.USDBRL.Buy)
	_, err = stmt.ExecContext(ctxPersist, convertPrice)
	if err != nil {
		return fmt.Errorf("erro ao inserir cotação no banco de dados: %w", err)
	}

	return nil
}
