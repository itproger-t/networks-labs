package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type DNSResponse struct {
	Domain    string    `json:"domain"`
	Command   string    `json:"command"`
	Result    string    `json:"result"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

func handleDNSShellExec() {
	http.HandleFunc("/dns", dnsHandler)
	fmt.Println("DNS Shell Exec сервер запущен на порту 8082")
	fmt.Println("Используйте: http://localhost:8082/dns?domain=google.com&type=A")
	http.ListenAndServe(":8082", nil)
}

func dnsHandler(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	queryType := r.URL.Query().Get("type")

	if domain == "" {
		http.Error(w, "Параметр 'domain' обязателен", http.StatusBadRequest)
		return
	}

	if queryType == "" {
		queryType = "A"
	}

	// Выполняем DNS запрос через shell команды
	result, err := executeDNSQuery(domain, queryType)

	response := DNSResponse{
		Domain:    domain,
		Timestamp: time.Now(),
	}

	if err != nil {
		response.Error = err.Error()
	} else {
		response.Result = result
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	fmt.Printf("DNS запрос для %s (тип: %s) выполнен\n", domain, queryType)
}

func executeDNSQuery(domain, queryType string) (string, error) {
	var cmd *exec.Cmd
	var cmdStr string

	// Выбираем команду в зависимости от ОС
	switch runtime.GOOS {
	case "windows":
		// Windows: используем nslookup
		cmdStr = fmt.Sprintf("nslookup -type=%s %s", queryType, domain)
		cmd = exec.Command("nslookup", "-type="+queryType, domain)
	case "linux", "darwin":
		// Linux/macOS: используем dig или nslookup
		if isCommandAvailable("dig") {
			cmdStr = fmt.Sprintf("dig %s %s +short", domain, queryType)
			cmd = exec.Command("dig", domain, queryType, "+short")
		} else {
			cmdStr = fmt.Sprintf("nslookup -type=%s %s", queryType, domain)
			cmd = exec.Command("nslookup", "-type="+queryType, domain)
		}
	default:
		return "", fmt.Errorf("неподдерживаемая ОС: %s", runtime.GOOS)
	}

	// Выполняем команду
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("ошибка выполнения команды '%s': %v", cmdStr, err)
	}

	result := strings.TrimSpace(string(output))

	if result == "" {
		result = "DNS запись не найдена"
	}

	return result, nil
}

// Вспомогательная функция для проверки доступности команды
func isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}
