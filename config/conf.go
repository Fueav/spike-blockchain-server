package config

import (
	"spike-blockchain-server/cache"
)

func Init() {
	//err := godotenv.Load("/root/go/src/github.com/spike-engine/spike-blockchain-server/.env")
	//if err != nil {
	//	panic(err)
	//}
	cache.Redis()
}
