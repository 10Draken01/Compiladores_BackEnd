package routes

import (
    "github.com/gin-gonic/gin"
    "api_compiladores/src/controllers"
    "go.mongodb.org/mongo-driver/mongo"
)

func UserRoute(router *gin.Engine, collection *mongo.Collection) {
    controllers.SetUserCollection(collection)

    userGroup := router.Group("/api/users")
    {
        userGroup.POST("/", controllers.CreateUser)
        userGroup.GET("/", controllers.GetUsers)
        userGroup.GET("/:id", controllers.GetUser)
        userGroup.PUT("/:id", controllers.UpdateUser)
        userGroup.DELETE("/:id", controllers.DeleteUser)
    }
}