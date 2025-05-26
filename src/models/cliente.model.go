package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Cliente struct {
    ID           primitive.ObjectID         `json:"id,omitempty" bson:"_id,omitempty"`
    Clave_Cliente any                       `json:"Clave_Cliente" bson:"Clave_Cliente"`
    Nombre       string                     `json:"Nombre" bson:"Nombre"`
    Celular      string                     `json:"Celular" bson:"Celular"`
    Email        string                     `json:"Email" bson:"Email"`
    Errores      map[string][]string        `json:"Errores" bson:"Errores"`
}
