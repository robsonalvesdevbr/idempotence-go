package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// Simulação de um registro de transações idempotentes
var transacoesIdempotentes = make(map[string]string)
var mu sync.Mutex

type Pagamento struct {
	Valor          float64 `json:"valor"`
	NumeroCartao   string  `json:"numero_cartao"`
	CVV            string  `json:"cvv"`
	DataValidade   string  `json:"data_validade"`
	Descricao      string  `json:"descricao"`
	IdempotencyKey string  // Não faz parte do body, mas do contexto da requisição
}

type RespostaPagamento struct {
	ID        string  `json:"id"`
	Status    string  `json:"status"`
	Valor     float64 `json:"valor"`
	Descricao string  `json:"descricao"`
}

func processarPagamento(pagamento Pagamento) (RespostaPagamento, error) {
	// Simulação do processamento de pagamento
	fmt.Printf("Processando pagamento de %.2f para '%s'\n", pagamento.Valor, pagamento.Descricao)
	// Aqui chamaria um gateway de pagamento real
	return RespostaPagamento{
		ID:        "TRANSACAO_" + fmt.Sprintf("%d", len(transacoesIdempotentes)+1),
		Status:    "SUCESSO",
		Valor:     pagamento.Valor,
		Descricao: pagamento.Descricao,
	}, nil
}

func criarPagamentoHandler(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		http.Error(w, "Idempotency-Key header is required", http.StatusBadRequest)
		return
	}

	mu.Lock()
	respostaExistente, existe := transacoesIdempotentes[idempotencyKey]
	mu.Unlock()

	if existe {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated) // Ou o status original da resposta
		fmt.Fprint(w, respostaExistente)
		fmt.Println("Requisição idempotente detectada. Retornando resposta anterior.")
		return
	}

	var pagamento Pagamento
	err := json.NewDecoder(r.Body).Decode(&pagamento)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	pagamento.IdempotencyKey = idempotencyKey

	resposta, err := processarPagamento(pagamento)
	if err != nil {
		http.Error(w, "Erro ao processar o pagamento", http.StatusInternalServerError)
		return
	}

	respostaJSON, err := json.Marshal(resposta)
	if err != nil {
		http.Error(w, "Erro ao serializar a resposta", http.StatusInternalServerError)
		return
	}

	mu.Lock()
	transacoesIdempotentes[idempotencyKey] = string(respostaJSON)
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, string(respostaJSON))
	fmt.Println("Pagamento processado e resposta armazenada com Idempotency-Key:", idempotencyKey)
}

func main() {
	http.HandleFunc("/pagamentos", criarPagamentoHandler)
	fmt.Println("Servidor rodando na porta 8000...")
	http.ListenAndServe(":8000", nil)
}
