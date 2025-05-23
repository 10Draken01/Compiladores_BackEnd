package routes

import (
    "github.com/gin-gonic/gin"
    "api_compiladores/src/controllers"
    "go.mongodb.org/mongo-driver/mongo"
)

func ClienteRoute(router *gin.Engine, collection *mongo.Collection) {
    controllers.SetClienteCollection(collection)

    clienteGroup := router.Group("/api/clientes")
    {
        clienteGroup.POST("/", controllers.CreateCliente)
        clienteGroup.GET("/page/:page", controllers.GetClientes)
        clienteGroup.GET("/:Clave_Cliente", controllers.GetCliente)
        clienteGroup.PUT("/:Clave_Cliente", controllers.UpdateCliente)
        clienteGroup.DELETE("/:Clave_Cliente", controllers.DeleteCliente)
    }
}