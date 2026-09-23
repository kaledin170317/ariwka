package env

import "os"

func GetServerHost() string {
	return os.Getenv("SERVER_HOST")
}
