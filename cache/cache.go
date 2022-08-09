package cache

import (
	"github.com/go-redis/redis"
	logger "github.com/ipfs/go-log"
	"spike-blockchain-server/constants"
)

var log = logger.Logger("cache")

var RedisClient *redis.Client

func Redis() error {
	client := redis.NewClient(
		&redis.Options{
			Addr:       constants.REDIS_ADDR,
			Password:   constants.REDIS_PW,
			MaxRetries: 1,
		})

	_, err := client.Ping().Result()

	if err != nil {
		// log connecting to redis failed
		log.Error("redis init err : ", err)
		return err
	}

	RedisClient = client
	return nil
}
