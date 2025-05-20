package controllers

import (
    "context"
    "net/http"
    "time"
	"log"

    "github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"

    "api_compiladores/src/models"
)

var userCollection *mongo.Collection

func SetUserCollection(c *mongo.Collection) {
    userCollection = c
}

func CreateUser(c *gin.Context) {
    var user models.User
    if err := c.ShouldBindJSON(&user); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    user.ID = primitive.NewObjectID()
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    _, err := userCollection.InsertOne(ctx, user)
    if err != nil {
		log.Println("Error al insertar usuario: ", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al insertar usuario"})
        return
    }

    c.JSON(http.StatusCreated, user)
}

func GetUsers(c *gin.Context) {
    var users []models.User
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    cursor, err := userCollection.Find(ctx, bson.M{})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer cursor.Close(ctx)

    for cursor.Next(ctx) {
        var user models.User
        if err := cursor.Decode(&user); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
        users = append(users, user)
    }

    c.JSON(http.StatusOK, users)
}

func GetUser(c *gin.Context) {
    id := c.Param("id")
    objID, _ := primitive.ObjectIDFromHex(id)

    var user models.User
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    err := userCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
        return
    }

    c.JSON(http.StatusOK, user)
}

func UpdateUser(c *gin.Context) {
    id := c.Param("id")
    objID, _ := primitive.ObjectIDFromHex(id)

    var user models.User
    if err := c.ShouldBindJSON(&user); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    update := bson.M{
        "$set": user,
    }

    _, err := userCollection.UpdateOne(ctx, bson.M{"_id": objID}, update)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar usuario"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Usuario actualizado"})
}

func DeleteUser(c *gin.Context) {
    id := c.Param("id")
    objID, _ := primitive.ObjectIDFromHex(id)

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    _, err := userCollection.DeleteOne(ctx, bson.M{"_id": objID})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar usuario"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Usuario eliminado"})
}
