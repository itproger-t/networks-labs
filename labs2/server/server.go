package main

import (
	"bufio"
	"fmt"
	"net"
)

func main() {
	// Запускаем сервер на localhost:8080
	listener, err := net.Listen("tcp", "localhost:8085")
	if err != nil {
		fmt.Println("Ошибка при запуске сервера:", err)
		return
	}
	defer listener.Close()
	fmt.Println("Сервер запущен на localhost:8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Ошибка при подключении клиента:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	fmt.Println("Клиент подключен:", conn.RemoteAddr())

	reader := bufio.NewReader(conn)
	for {
		// Читаем строку от клиента
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Ошибка чтения:", err)
			return
		}
		fmt.Printf("Получено от клиента: %s", message)

		// Отправляем обратно клиенту
		_, err = conn.Write([]byte(message))
		if err != nil {
			fmt.Println("Ошибка отправки:", err)
			return
		}
	}
}
