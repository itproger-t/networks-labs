package main

import (
	"fmt"
	"net"
	"strings"
	"time"
)

func handleHTTPRawSocket() {
	// Создаем TCP сокет
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Printf("Ошибка создания сокета: %v\n", err)
		return
	}
	defer listener.Close()

	fmt.Println("HTTP сервер запущен на порту 8080 (Raw Socket)")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Ошибка принятия соединения: %v\n", err)
			continue
		}

		go handleHTTPConnection(conn)
	}
}

func handleHTTPConnection(conn net.Conn) {
	defer conn.Close()

	// Читаем запрос
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Printf("Ошибка чтения запроса: %v\n", err)
		return
	}

	request := string(buffer[:n])
	fmt.Printf("Получен HTTP запрос:\n%s\n", request)

	// Парсим первую строку запроса
	lines := strings.Split(request, "\r\n")
	if len(lines) == 0 {
		return
	}

	requestLine := strings.Split(lines[0], " ")
	if len(requestLine) < 3 {
		return
	}

	method := requestLine[0]
	path := requestLine[1]

	// Формируем HTTP ответ
	responseBody := fmt.Sprintf(`
    <html>
    <head><title>Raw Socket HTTP Server</title></head>
    <body>
        <h1>Hello from Raw Socket HTTP Server!</h1>
        <p>Method: %s</p>
        <p>Path: %s</p>
        <p>Time: %s</p>
    </body>
    </html>`, method, path, time.Now().Format("2006-01-02 15:04:05"))

	httpResponse := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\n"+
			"Content-Type: text/html; charset=utf-8\r\n"+
			"Content-Length: %d\r\n"+
			"Connection: close\r\n"+
			"\r\n%s",
		len(responseBody), responseBody)

	// Отправляем ответ
	conn.Write([]byte(httpResponse))
}
