package main

import (
	"log"
	"os"

	"bookcabin-flight/internal/logging"
	"bookcabin-flight/internal/server"
)

func main() {
	logger, err := logging.Open(logDirectory())
	if err != nil {
		log.Fatalf("logging failed: %v", err)
	}
	defer func() {
		if err := logger.Close(); err != nil {
			log.Printf("closing the logs failed: %v", err)
		}
	}()

	if err := server.Run(":"+port(), server.NewRouter(logger)); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func port() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return "8080"
}

// logDirectory is where the logs are written, a logs directory of the working
// directory unless LOG_DIR says otherwise.
func logDirectory() string {
	if dir := os.Getenv("LOG_DIR"); dir != "" {
		return dir
	}

	return "logs"
}
