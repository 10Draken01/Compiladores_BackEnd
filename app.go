package main

import (
    "log"

    "api_compiladores/src/db" // <- Reemplaza "project-root" con el nombre real de tu módulo
)

func main() {
    db.InitMongoDB()

    // Puedes usar db.MongoDatabase aquí para acceder a colecciones
    log.Println("Aplicación iniciada.")
}
