package controllers

import (
	"context"
	"log"
	"net/http"
	"math"
	"fmt"
	"regexp"
	"time"
	"strconv"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"api_compiladores/src/models"
	"api_compiladores/src/utils"
)

type ExampleClienteCreate struct {
	Clave_Cliente string `json:"Clave_Cliente" example:"001"`
	Nombre        string `json:"Nombre" example:"Pedro"`
	Celular       string `json:"Celular" example:"9613214782"`
	Email         string `json:"Email" example:"correo@example.com"`
}

type ExampleClientePut struct {
	Nombre  string `json:"Nombre" example:"Pedro"`
	Celular string `json:"Celular" example:"9613214782"`
	Email   string `json:"Email" example:"correo@example.com"`
}

var (
	// Expresion regular para validar que el campo Clave_Cliente sea un número entero
	identRegexNumeric = regexp.MustCompile(`^[0-9]*$`);
	// Expresion regular para validar que el campo Nombre sea una cadena de caracteres sin simbolos, numeros y 1 espacio entre palabras
	identRegexNombre = regexp.MustCompile(`^[a-zA-ZáéíóúÁÉÍÓÚñÑüÜ\s]+$`)
	// // Expresion regular para validar que el campo Celular sea un número de 10 dígitos
	identRegexCelular = regexp.MustCompile(`^(91[6-9]|93[24]|96[1-8]|99[24])\d{7}$`)
	// // Expresion regular para validar que el campo Email sea un correo electronico valido
	identRegexEmail = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@(gmail\.com|hotmail\.com|yahoo\.com|outlook\.com|live\.com|icloud\.com|protonmail\.com|aol\.com|msn\.com|gmx\.com|ymail\.com|me\.com|mail\.com|zoho\.com|edu\.mx|edu\.com|edu\.org)$`)
)

var clienteCollection *mongo.Collection

var exampleCreate = ExampleClienteCreate{
	Clave_Cliente: "001",
	Nombre:        "Pedro",
	Celular:       "9613214782",
	Email:         "correo@example.com",
}

var examplePut = ExampleClientePut{
	Nombre:  "Pedro",
	Celular: "9613214782",
	Email:   "correo@example.com",
}

func SetClienteCollection(c *mongo.Collection) {
	clienteCollection = c
}

func CreateCliente(c *gin.Context) {
	var cliente models.Cliente
	if err := c.ShouldBindJSON(&cliente); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if cliente.Clave_Cliente == nil {
		sendError(c, "Clave_Cliente es obligatorio")
		return
	}

	claveCliente, err := normalizeClaveCliente(cliente.Clave_Cliente)
	if err != nil {
		sendError(c, err.Error())
		return
	}

	cliente.ID = primitive.NewObjectID()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verificar existencia previa
	var existingCliente models.Cliente
	err = clienteCollection.FindOne(ctx, bson.M{"Clave_Cliente": claveCliente}).Decode(&existingCliente)
	if err == nil {
		message := fmt.Sprintf("El cliente con Clave_Cliente %s ya existe", claveCliente)
		log.Println("error:", message)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": message,
		})
		return
	} else if err != mongo.ErrNoDocuments {
		log.Println("Error al verificar existencia de cliente:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al verificar existencia de cliente"})
		return
	}

	cliente.Clave_Cliente = claveCliente
	utils.ValidateCliente(&cliente)

	if _, err := clienteCollection.InsertOne(ctx, cliente); err != nil {
		log.Println("Error al insertar cliente:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al insertar cliente"})
		return
	}

	c.JSON(http.StatusCreated, cliente)
}

// Función para validar y normalizar Clave_Cliente
func normalizeClaveCliente(clave interface{}) (string, error) {
	switch v := clave.(type) {
	case string:
		if !identRegexNumeric.MatchString(v) {
			return "", fmt.Errorf("Clave_Cliente no es un identificador válido")
		}
		_, err := strconv.Atoi(v)
		if err != nil {
			return "", fmt.Errorf("Clave_Cliente no es un número válido")
		}
		return v, nil

	case float64:
		if math.Mod(v, 1) != 0 {
			return "", fmt.Errorf("Clave_Cliente no puede ser decimal")
		}
		if v < 0 {
			return "", fmt.Errorf("Clave_Cliente no puede ser negativo")
		}
		return fmt.Sprintf("%d", int(v)), nil

	default:
		return "", fmt.Errorf("Tipo de dato no válido para Clave_Cliente")
	}
}

// Función para enviar errores con log y ejemplo
func sendError(c *gin.Context, message string) {
	log.Println("error:", message)
	c.JSON(http.StatusBadRequest, gin.H{
		"error":   message,
		"example": exampleCreate,
	})
}

func GetClientes(c *gin.Context) {
	page := c.Param("page")

	if page == "" {
		message := "Error: El atributo page es obligatorio | debe ser un número entero del 1 al 10000"
		log.Println(message)
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}

	if !identRegexNumeric.MatchString(page) {
		message := "Error: " + page + " no es un identificador válido"
		log.Println(message)
		jsonData := gin.H{
			"error":   message,
			"example": "1",
		}
		c.JSON(http.StatusBadRequest, jsonData)
		return
	}

	// Convertir a entero
	intPage, err := strconv.Atoi(page)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Page debe ser un número válido"})
		return
	}

	// Validar rango de página
	if intPage < 1 || intPage > 10000 {
		message := "Error: " + page + " no es un número válido | debe ser un número entero del 1 al 10000"
		log.Println(message)
		jsonData := gin.H{
			"error":   message,
			"example": "1",
		}
		c.JSON(http.StatusBadRequest, jsonData)
		return
	}

	limit := 100
	skip := (intPage - 1) * limit

	var clientes []models.Cliente

	cacheKey := fmt.Sprintf("clientes_cache_page_%d", intPage)
	cachedClientes, err := utils.RedisClient.Get(utils.Ctx, cacheKey).Result()
	if err == nil {
		if err := json.Unmarshal([]byte(cachedClientes), &clientes); err == nil {
			// Si los datos están en caché, los devolvemos
			log.Println("Datos obtenidos de la caché")
			c.JSON(http.StatusOK, clientes)
			return
		}
		log.Println("Error al deserializar datos de caché: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al deserializar datos de caché"})
		return
	}

	// Si no están en caché, los obtenemos de la base de datos
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	options := options.Find()
	options.SetLimit(int64(limit))
	options.SetSkip(int64(skip))

	cursor, err := clienteCollection.Find(ctx, bson.M{}, options)
	if err != nil {
		log.Println("Error al obtener clientes: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var cliente models.Cliente
		if err := cursor.Decode(&cliente); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		clientes = append(clientes, cliente)
	}

	log.Println("Datos obtenidos de la base de datos")
	jsonData, _ := json.Marshal(clientes)
	utils.RedisClient.Set(utils.Ctx, cacheKey, jsonData, 5*time.Minute)
	c.JSON(http.StatusOK, clientes)
}

func GetCliente(c *gin.Context) {
	Clave_Cliente := c.Param("Clave_Cliente")

	var cliente models.Cliente
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := clienteCollection.FindOne(ctx, bson.M{"Clave_Cliente": Clave_Cliente}).Decode(&cliente)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cliente no encontrado"})
		return
	}

	c.JSON(http.StatusOK, cliente)
}

func UpdateCliente(c *gin.Context) {
	Clave_Cliente := c.Param("Clave_Cliente")
	if Clave_Cliente == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El campo Clave_Cliente es obligatorio"})
		return
	}

	var cliente models.Cliente
	if err := c.ShouldBindJSON(&cliente); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	utils.ValidateCliente(&cliente)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Solo actualizamos los campos válidos, no el ID
	update := bson.M{
		"$set": bson.M{
			"Nombre":  cliente.Nombre,
			"Celular": cliente.Celular,
			"Email":   cliente.Email,
			"Errores": cliente.Errores,
		},
	}

	result, err := clienteCollection.UpdateOne(ctx, bson.M{"Clave_Cliente": Clave_Cliente}, update)
	if err != nil {
		log.Println("Error al actualizar cliente: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar cliente"})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cliente no encontrado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cliente actualizado correctamente"})
}

func DeleteCliente(c *gin.Context) {
	Clave_Cliente := c.Param("Clave_Cliente")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := clienteCollection.DeleteOne(ctx, bson.M{"Clave_Cliente": Clave_Cliente})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar cliente"})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cliente no encontrado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cliente eliminado correctamente"})
}