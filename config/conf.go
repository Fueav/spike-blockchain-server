package config

import (
	"github.com/joho/godotenv"

	"spike-blockchain-server/cache"
)

func Init() {
	err := godotenv.Load("/root/go/src/github.com/spike-engine/spike-blockchain-server/.env")
	if err != nil {
		panic(err)
	}
	cache.Redis()
}
