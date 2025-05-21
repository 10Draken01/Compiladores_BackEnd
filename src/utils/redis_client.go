package utils

import (
    "context"
    "github.com/go-redis/redis/v8"
)

var Ctx = context.Background()

var RedisClient *redis.Client

func ConnectRedis() {
    RedisClient = redis.NewClient(&redis.Options{
        Addr:     "localhost:6379", // Dirección del contenedor Redis
        Password: "",               // Sin contraseña por ahora
        DB:       0,                // Base de datos 0
    })
}
