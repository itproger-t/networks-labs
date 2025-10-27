package protocol

type Message struct {
	Type     string   `json:"type"`     // "broadcast", "private", "group", "file", "group_create"
	Sender   string   `json:"sender"`   // ID или ник клиента
	Target   string   `json:"target"`   // имя получателя или группы
	Targets  []string `json:"targets"`  // для создания групп
	Group    string   `json:"group"`    // имя группы (если group)
	Content  string   `json:"content"`  // текст сообщения
	FileName string   `json:"filename"` // имя файла, если передаём файл
	Data     []byte   `json:"data"`     // содержимое файла (base64)
}
