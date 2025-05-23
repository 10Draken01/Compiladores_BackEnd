// utils/agregarUsers.go
package utils

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jaswdr/faker"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"api_compiladores/src/models"
)

func AddOneHundredThousandUsers(collection *mongo.Collection) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	count, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Fatal("Error al contar usuarios:", err)
	}

	if count > 0 {
		fmt.Println("Ya existen usuarios en la base de datos.")
		return
	}

	fmt.Println("Insertando usuarios...")
	f := faker.New()
	var users []interface{}

	for i := 1; i <= 20000000; i++ {
		user := models.Cliente{
			ID:            primitive.NewObjectID(),
			Clave_Cliente: fmt.Sprintf("%010d", i),
			Nombre:        f.Person().Name(),
			Celular:       f.Phone().Number(),
			Email:         f.Internet().Email(),
		}
		users = append(users, user)

		// Inserta en lotes de 1000
		if i%1000 == 0 {
			_, err := collection.InsertMany(ctx, users)
			if err != nil {
				log.Fatal("Error al insertar usuarios:", err)
			}
			users = users[:0] // Reiniciar slice
			fmt.Printf("Insertados %d usuarios...\n", i)
		}
	}
}
