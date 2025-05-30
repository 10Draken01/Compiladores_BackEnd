package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// User estructura para mapear los documentos de la colección users
type User struct {
	ID           interface{} `bson:"_id,omitempty"`
	ClaveCliente string      `bson:"Clave_Cliente"`
	// Agrega aquí los demás campos que tengas en tu colección
}

func main() {
	// URI de conexión a MongoDB
	uri := "mongodb://localhost:27017"
	
	// Crear contexto con timeout más largo
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Crear cliente de MongoDB con opciones mejoradas
	clientOptions := options.Client().
		ApplyURI(uri).
		SetConnectTimeout(30 * time.Second).
		SetSocketTimeout(30 * time.Second).
		SetServerSelectionTimeout(30 * time.Second).
		SetMaxPoolSize(10).
		SetMinPoolSize(1)
	
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal("Error creando/conectando cliente MongoDB:", err)
	}
	defer client.Disconnect(ctx)

	// Verificar conexión con timeout extendido
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer pingCancel()
	
	err = client.Ping(pingCtx, nil)
	if err != nil {
		log.Fatal("Error haciendo ping a MongoDB. Verifica que MongoDB esté ejecutándose:", err)
	}
	fmt.Println("Conectado exitosamente a MongoDB")

	// Seleccionar base de datos y colección
	db := client.Database("lexicodb")
	collection := db.Collection("users")
	
	// Verificar que la colección existe
	collections, err := db.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		log.Fatal("Error listando colecciones:", err)
	}
	
	fmt.Printf("Colecciones disponibles: %v\n", collections)
	
	// Verificar si la colección 'users' existe
	collectionExists := false
	for _, col := range collections {
		if col == "users" {
			collectionExists = true
			break
		}
	}
	
	if !collectionExists {
		fmt.Println("Advertencia: La colección 'users' no existe en la base de datos 'lexicodb'")
		return
	}

	// Parámetros de paginación
	pageSize := int64(100)
	pageNumber := int64(1) // Página inicial (puedes cambiar esto)

	// Obtener datos con paginación
	users, totalDocuments, err := getUsersWithPagination(ctx, collection, pageNumber, pageSize)
	if err != nil {
		log.Fatal("Error obteniendo usuarios:", err)
	}

	// Mostrar resultados
	fmt.Printf("Total de documentos: %d\n", totalDocuments)
	fmt.Printf("Página: %d, Tamaño: %d\n", pageNumber, pageSize)
	fmt.Printf("Usuarios obtenidos: %d\n", len(users))
	
	// Mostrar algunos usuarios de ejemplo
	for i, user := range users {
		if i < 5 { // Mostrar solo los primeros 5
			fmt.Printf("Usuario %d - ID: %v, Clave_Cliente: %s\n", i+1, user.ID, user.ClaveCliente)
		}
	}
	
	if len(users) > 5 {
		fmt.Printf("... y %d usuarios más\n", len(users)-5)
	}
}

// getUsersWithPagination obtiene usuarios con paginación optimizada usando el índice Clave_Cliente
func getUsersWithPagination(ctx context.Context, collection *mongo.Collection, page, limit int64) ([]User, int64, error) {
	// Calcular skip
	skip := (page - 1) * limit

	// Opciones de búsqueda con paginación
	findOptions := options.Find()
	findOptions.SetLimit(limit)
	findOptions.SetSkip(skip)
	// Ordenar por Clave_Cliente para aprovechar el índice
	findOptions.SetSort(bson.D{{Key: "Clave_Cliente", Value: 1}})

	// Ejecutar consulta
	cursor, err := collection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	// Decodificar resultados
	var users []User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, 0, err
	}

	// Obtener total de documentos
	totalCount, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	return users, totalCount, nil
}

// getUsersWithCursorPagination alternativa usando cursor-based pagination (más eficiente para grandes datasets)
func getUsersWithCursorPagination(ctx context.Context, collection *mongo.Collection, lastClaveCliente string, limit int64) ([]User, error) {
	filter := bson.M{}
	
	// Si tenemos un cursor (última Clave_Cliente), buscar desde ahí
	if lastClaveCliente != "" {
		filter["Clave_Cliente"] = bson.M{"$gt": lastClaveCliente}
	}

	findOptions := options.Find()
	findOptions.SetLimit(limit)
	findOptions.SetSort(bson.D{{Key: "Clave_Cliente", Value: 1}})

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, err
	}

	return users, nil
}

// Función auxiliar para obtener información de paginación
func getPaginationInfo(totalDocuments, pageSize, currentPage int64) map[string]int64 {
	totalPages := (totalDocuments + pageSize - 1) / pageSize
	
	return map[string]int64{
		"totalDocuments": totalDocuments,
		"totalPages":     totalPages,
		"currentPage":    currentPage,
		"pageSize":       pageSize,
		"hasNextPage":    boolToInt64(currentPage < totalPages),
		"hasPrevPage":    boolToInt64(currentPage > 1),
	}
}

func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}