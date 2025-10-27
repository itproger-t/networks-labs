package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	// Подключаемся к серверу
	conn, err := net.Dial("tcp", "localhost:8085")
	if err != nil {
		fmt.Println("Ошибка подключения к серверу:", err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)
	serverReader := bufio.NewReader(conn)

	fmt.Println("Введите текст (для выхода введите 'exit'):")

	for {
		fmt.Print("> ")
		text, _ := reader.ReadString('\n')

		if text == "exit\n" {
			break
		}

		// Отправляем текст серверу
		_, err := conn.Write([]byte(text))
		if err != nil {
			fmt.Println("Ошибка отправки:", err)
			return
		}

		// Читаем ответ от сервера
		response, err := serverReader.ReadString('\n')
		if err != nil {
			fmt.Println("Ошибка чтения ответа:", err)
			return
		}

		fmt.Println("Ответ от сервера:", response)
	}
}
