package main

import (
	"fmt"
	"sync"
)

func main() {
	var wg sync.WaitGroup

	// Запускаем все три сервера в отдельных горутинах
	wg.Add(3)

	go func() {
		defer wg.Done()
		handleHTTPRawSocket()
	}()

	go func() {
		defer wg.Done()
		handleFileProtocol()
	}()

	go func() {
		defer wg.Done()
		handleDNSShellExec()
	}()

	fmt.Println("Все серверы запущены:")
	fmt.Println("1. HTTP Socket - http://localhost:8080")
	fmt.Println("2. File Protocol - http://localhost:8081/file?path=file:///path/to/file")
	fmt.Println("3. DNS Shell Exec - http://localhost:8082/dns?domain=google.com&type=A")

	wg.Wait()
}
