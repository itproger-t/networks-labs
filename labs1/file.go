package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func handleFileProtocol() {
	http.HandleFunc("/file", fileHandler)
	fmt.Println("File протокол сервер запущен на порту 8081")
	fmt.Println("Используйте: http://localhost:8081/file?path=file:///path/to/file")
	http.ListenAndServe(":8081", nil)
}

func fileHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем параметр path из запроса
	pathParam := r.URL.Query().Get("path")
	if pathParam == "" {
		http.Error(w, "Параметр 'path' обязателен", http.StatusBadRequest)
		return
	}

	// Парсим URL с file протоколом
	parsedURL, err := url.Parse(pathParam)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка парсинга URL: %v", err), http.StatusBadRequest)
		return
	}

	if parsedURL.Scheme != "file" {
		http.Error(w, "Поддерживается только протокол file://", http.StatusBadRequest)
		return
	}

	// Получаем путь к файлу
	filePath := parsedURL.Path

	// Для Windows нужно убрать первый слэш
	if len(filePath) > 0 && filePath[0] == '/' && len(filePath) > 1 && filePath[2] == ':' {
		filePath = filePath[1:]
	}

	// Проверяем существование файла
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, fmt.Sprintf("Файл не найден: %s", filePath), http.StatusNotFound)
		return
	}

	// Читаем содержимое файла
	content, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка чтения файла: %v", err), http.StatusInternalServerError)
		return
	}

	// Определяем MIME тип по расширению
	ext := strings.ToLower(filepath.Ext(filePath))
	contentType := "text/plain"

	switch ext {
	case ".html", ".htm":
		contentType = "text/html"
	case ".css":
		contentType = "text/css"
	case ".js":
		contentType = "application/javascript"
	case ".json":
		contentType = "application/json"
	case ".xml":
		contentType = "application/xml"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filepath.Base(filePath)))

	w.Write(content)

	fmt.Printf("Обработан запрос файла: %s\n", filePath)
}
