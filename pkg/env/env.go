package env

import "os"

func GetServerHost() string {
	return os.Getenv("SERVER_HOST")
}

func GetJWTSecret() string {
	return os.Getenv("JWT_SECRET")
}
