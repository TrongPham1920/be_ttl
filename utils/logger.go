package utils

import (
	"fmt"
	"log"
	"os"
	"time"
)

var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

func init() {
	// Tạo thư mục logs nếu chưa tồn tại
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Fatal(err)
	}

	// Tạo file log với timestamp
	timestamp := time.Now().Format("2006-01-02")
	logFile, err := os.OpenFile(fmt.Sprintf("logs/app-%s.log", timestamp), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}

	// Khởi tạo loggers
	InfoLogger = log.New(logFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(logFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// LogInfo ghi log thông tin
func LogInfo(format string, v ...interface{}) {
	InfoLogger.Printf(format, v...)
}

// LogError ghi log lỗi
func LogError(format string, v ...interface{}) {
	ErrorLogger.Printf(format, v...)
}
