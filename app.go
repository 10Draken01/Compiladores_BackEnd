package main

import (
    "github.com/gin-gonic/gin"
    "github.com/joho/godotenv"
    "log"
    "os"
    "api_compiladores/src/config"
    "api_compiladores/src/routes"
    "api_compiladores/src/utils"
	"github.com/gin-contrib/cors"
)

func validationENV(env string, envDefault string) string {
    if env == "" {
        return envDefault
    }
    return env
}

func main() {
    // Conectar a Redis
    utils.ConnectRedis()

    err := godotenv.Load()
    if err != nil {
        log.Println("Error cargando el archivo .env:", err)
    }

	// Leer puerto del .env
	port := validationENV(os.Getenv("PORT"), "8000")

    uri := validationENV(os.Getenv("MONGO_URI"), "mongodb://localhost:27017")

    dbName := validationENV(os.Getenv("DB_NAME"), "lexicodb")

    config.ConnectDB(uri)

    r := gin.Default()

    // Habilitar CORS
    r.Use(cors.Default())

    userCollection := config.GetCollection(dbName, "users")
    routes.UserRoute(r, userCollection)

    r.Run(":" + port)
}