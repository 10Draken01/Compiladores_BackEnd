package db

import (
    "context"
    "log"
    "os"
    "time"

    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "github.com/joho/godotenv"
)

var MongoClient *mongo.Client
var MongoDatabase *mongo.Database

func InitMongoDB() {
    err := godotenv.Load()
    if err != nil {
        log.Fatal("Error cargando el archivo .env:", err)
    }

    uri := os.Getenv("MONGO_URI")
    dbName := os.Getenv("DB_NAME")

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    clientOptions := options.Client().ApplyURI(uri)
    client, err := mongo.Connect(ctx, clientOptions)
    if err != nil {
        log.Fatal("Error al conectar a MongoDB:", err)
    }

    if err := client.Ping(ctx, nil); err != nil {
        log.Fatal("MongoDB no responde:", err)
    }

    MongoClient = client
    MongoDatabase = client.Database(dbName)

    log.Println("✅ Conexión a MongoDB exitosa")
}
