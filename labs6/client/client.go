package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	. "tcp-chat/protocol"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:9000")
	if err != nil {
		fmt.Println("Ошибка подключения:", err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Введите имя: ")
	name, _ := reader.ReadString('\n')
	conn.Write([]byte(name))

	go listenServer(conn)

	for {
		fmt.Print("> ")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)

		if strings.HasPrefix(text, "/group create") {
			parts := strings.Split(text, " ")
			if len(parts) < 4 {
				fmt.Println("Использование: /group create <group> <user1> <user2> ...")
				continue
			}
			groupName := parts[2]
			users := parts[3:]
			msg := Message{
				Type:    "group_create",
				Sender:  name,
				Group:   groupName,
				Targets: users,
			}
			data, _ := json.Marshal(msg)
			conn.Write(append(data, '\n'))
		} else if strings.HasPrefix(text, "/group send") {
			parts := strings.SplitN(text, " ", 4)
			if len(parts) < 4 {
				fmt.Println("Использование: /group send <group> <текст>")
				continue
			}
			groupName := parts[2]
			content := parts[3]
			msg := Message{
				Type:    "group",
				Sender:  name,
				Group:   groupName,
				Content: content,
			}
			data, _ := json.Marshal(msg)
			conn.Write(append(data, '\n'))
		} else if strings.HasPrefix(text, "/sendfile") {
			parts := strings.SplitN(text, " ", 3)
			if len(parts) < 3 {
				fmt.Println("Использование: /sendfile <user|group|all> <путь>")
				continue
			}
			target := parts[1]
			path := parts[2]

			data, err := os.ReadFile(path)
			if err != nil {
				fmt.Println("Ошибка чтения файла:", err)
				continue
			}

			msg := Message{
				Type:     "file",
				Sender:   name,
				FileName: filepath.Base(path),
				Data:     data,
			}

			// Кому отправляем: одному, группе или всем
			if target == "all" {
			} else if strings.HasPrefix(target, "group:") {
				msg.Group = strings.TrimPrefix(target, "group:")
			} else {
				msg.Target = target
			}

			out, _ := json.Marshal(msg)
			conn.Write(append(out, '\n'))
		} else if strings.HasPrefix(text, "/pm") {
			parts := strings.SplitN(text, " ", 3)
			if len(parts) < 3 {
				fmt.Println("Использование: /pm <user> <текст>")
				continue
			}
			target := parts[1]
			content := parts[2]
			msg := Message{
				Type:    "private",
				Sender:  name,
				Target:  target,
				Content: content,
			}
			data, _ := json.Marshal(msg)
			conn.Write(append(data, '\n'))

		} else {
			msg := Message{
				Type:    "broadcast",
				Sender:  name,
				Content: text,
			}
			data, _ := json.Marshal(msg)
			conn.Write(append(data, '\n'))
		}

	}

}

func listenServer(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		msgBytes, err := reader.ReadBytes('\n')
		if err != nil {
			fmt.Println("Соединение закрыто")
			os.Exit(0)
		}

		var msg Message
		json.Unmarshal(msgBytes, &msg)

		switch msg.Type {
		case "update":
			fmt.Println("Список клиентов:", msg.Content)
		case "broadcast":
			fmt.Printf("[Общий чат][%s]: %s\n", msg.Sender, msg.Content)
		case "private":
			fmt.Printf("[ЛС от %s]: %s\n", msg.Sender, msg.Content)
		case "group":
			fmt.Printf("[Группа %s][%s]: %s\n", msg.Group, msg.Sender, msg.Content)
		case "info":
			fmt.Println("[Сервер]:", msg.Content)
		case "file":
			os.MkdirAll("downloads", 0755)

			filePath := filepath.Join("/Users/magamadov/Downloads/new", msg.FileName)
			err := os.WriteFile(filePath, msg.Data, 0644)
			if err != nil {
				fmt.Println("Ошибка сохранения файла:", err)
			} else {
				fmt.Printf("[Файл от %s]: %s сохранён в %s\n", msg.Sender, msg.FileName, filePath)
			}

		}

	}
}
