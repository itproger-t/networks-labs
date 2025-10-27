package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	. "tcp-chat/protocol"
)

type Client struct {
	Conn net.Conn
	Name string
}

var (
	clients = make(map[string]Client)
	groups  = make(map[string][]string) // группа -> список участников
	mu      sync.Mutex
)

func main() {
	listener, err := net.Listen("tcp", ":9000")
	if err != nil {
		fmt.Println("Ошибка запуска сервера:", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Println("Сервер запущен на :9000")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Ошибка подключения:", err)
			continue
		}
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	mu.Lock()
	if _, exists := clients[name]; exists {
		mu.Unlock()
		conn.Write([]byte("Имя занято, выберите другое\n"))
		conn.Close()
		return
	}
	clients[name] = Client{Conn: conn, Name: name}
	mu.Unlock()

	broadcastClientList()

	for {
		msgBytes, err := reader.ReadBytes('\n')
		if err != nil {
			fmt.Printf("Клиент %s отключился\n", name)
			mu.Lock()
			delete(clients, name)
			mu.Unlock()
			broadcastClientList()
			return
		}

		var msg Message
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			fmt.Println("Ошибка парсинга сообщения:", err)
			continue
		}

		routeMessage(msg)
	}
}

func routeMessage(msg Message) {
	switch msg.Type {
	case "broadcast":
		sendToAll(msg)
	case "private":
		sendToClient(msg.Target, msg)
	case "group":
		sendToGroup(msg.Group, msg)
	case "file":
		if msg.Target != "" {
			sendToClient(msg.Target, msg)
		} else if msg.Group != "" {
			sendToGroup(msg.Group, msg)
		} else {
			sendToAll(msg)
		}
	case "group_create":
		mu.Lock()
		if _, exists := groups[msg.Group]; exists {
			mu.Unlock()
			notify := Message{
				Type:    "info",
				Content: fmt.Sprintf("Группа %s уже существует", msg.Group),
			}
			sendToClient(msg.Sender, notify)
			return
		}
		members := append(msg.Targets, msg.Sender)
		groups[msg.Group] = members
		mu.Unlock()

		notify := Message{
			Type:    "info",
			Content: fmt.Sprintf("Группа %s создана (%v)", msg.Group, members),
		}
		sendToAll(notify)

	}
}

func sendToAll(msg Message) {
	mu.Lock()
	defer mu.Unlock()
	for _, cl := range clients {
		sendJSON(cl.Conn, msg)
	}
}

func sendToClient(name string, msg Message) {
	mu.Lock()
	defer mu.Unlock()
	if cl, ok := clients[name]; ok {
		sendJSON(cl.Conn, msg)
	}
}

func sendToGroup(group string, msg Message) {
	mu.Lock()
	defer mu.Unlock()
	if members, ok := groups[group]; ok {
		for _, name := range members {
			if cl, ok := clients[name]; ok {
				sendJSON(cl.Conn, msg)
			}
		}
	}
}

func broadcastClientList() {
	mu.Lock()
	defer mu.Unlock()

	list := []string{}
	for name := range clients {
		list = append(list, name)
	}

	update := Message{Type: "update", Content: fmt.Sprintf("%v", list)}
	for _, cl := range clients {
		sendJSON(cl.Conn, update)
	}
}

func sendJSON(conn net.Conn, msg Message) {
	data, _ := json.Marshal(msg)
	conn.Write(append(data, '\n'))
}
