package config

import (
    "context"
    "fmt"
    "log"
    "time"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "go.mongodb.org/mongo-driver/mongo/readpref"
)

var DB *mongo.Client

func ConnectDB(uri string) {
    // Crear contexto con timeout más largo
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Configuración mejorada del cliente con timeouts específicos
    clientOpts := options.Client().
        ApplyURI(uri).
        SetConnectTimeout(30 * time.Second).
        SetSocketTimeout(30 * time.Second).
        SetServerSelectionTimeout(30 * time.Second).
        SetMaxPoolSize(10).
        SetMinPoolSize(1)

    client, err := mongo.Connect(ctx, clientOpts)
    if err != nil {
        log.Fatalf("Error al conectar a MongoDB: %v", err)
    }

    // Verificar conexión con ping y timeout extendido
    pingCtx, pingCancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer pingCancel()
    
    err = client.Ping(pingCtx, readpref.Primary())
    if err != nil {
        log.Fatalf("No se pudo hacer ping a MongoDB. Verifica que MongoDB esté ejecutándose: %v", err)
    }

    log.Println("Conexión a MongoDB establecida correctamente")
    DB = client
}

func GetCollection(dbName string, collectionName string) *mongo.Collection {
    return DB.Database(dbName).Collection(collectionName)
}

// Función auxiliar para verificar si una colección existe
func CollectionExists(dbName string, collectionName string) (bool, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    db := DB.Database(dbName)
    collections, err := db.ListCollectionNames(ctx, bson.M{})
    if err != nil {
        return false, fmt.Errorf("error listando colecciones: %v", err)
    }

    for _, col := range collections {
        if col == collectionName {
            return true, nil
        }
    }
    
    return false, nil
}

// Función auxiliar para obtener información de la base de datos
func GetDatabaseInfo(dbName string) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    db := DB.Database(dbName)
    collections, err := db.ListCollectionNames(ctx, bson.M{})
    if err != nil {
        log.Printf("Error listando colecciones: %v", err)
        return
    }

    fmt.Printf("Base de datos '%s' - Colecciones disponibles: %v\n", dbName, collections)
}

// Función para cerrar la conexión de manera segura
func DisconnectDB() {
    if DB != nil {
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        
        if err := DB.Disconnect(ctx); err != nil {
            log.Printf("Error al desconectar de MongoDB: %v", err)
        } else {
            log.Println("Desconectado de MongoDB correctamente")
        }
    }
}