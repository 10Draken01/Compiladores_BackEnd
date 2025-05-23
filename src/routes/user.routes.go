package routes

import (
    "github.com/gin-gonic/gin"
    "api_compiladores/src/controllers"
    "go.mongodb.org/mongo-driver/mongo"
)

func UserRoute(router *gin.Engine, collection *mongo.Collection) {
    controllers.SetUserCollection(collection)

    userGroup := router.Group("/api/clientes")
    {
        userGroup.POST("/", controllers.CreateUser)
        userGroup.GET("/page/:page", controllers.GetUsers)
        userGroup.GET("/:Clave_Cliente", controllers.GetUser)
        userGroup.PUT("/:Clave_Cliente", controllers.UpdateUser)
        userGroup.DELETE("/:Clave_Cliente", controllers.DeleteUser)
    }
}