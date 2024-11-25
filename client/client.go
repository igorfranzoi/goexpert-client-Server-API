package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const LOCAL_HOST = "http://localhost:3030/cotacao"

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	valor, err := getServerPrice(ctx)
	if err != nil {
		log.Fatalf("erro [GET-client] - cotação: %v", err)
	}

	if err := writeInfoFile(valor); err != nil {
		log.Fatalf("problema ao salvar no arquivo: %v", err)
	}

	fmt.Println("Cotação salva com sucesso!")
}

func getServerPrice(ctx context.Context) (string, error) {
	client := &http.Client{Timeout: 300 * time.Millisecond}

	req, err := http.NewRequestWithContext(ctx, "GET", LOCAL_HOST, nil)
	if err != nil {
		return "", fmt.Errorf("erro ao criar requisição: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("erro ao fazer requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("erro na resposta do servidor: %s", resp.Status)
	}

	var bodyResponse map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&bodyResponse); err != nil {
		return "", fmt.Errorf("erro ao realizar o parse da resposta: %w", err)
	}

	return bodyResponse["bid"], nil
}

func writeInfoFile(valor string) error {
	content := fmt.Sprintf("dolar: %s", valor)
	return os.WriteFile("cotacao.txt", []byte(content), 0644)
}
