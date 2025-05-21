package controllers

import (
	"context"
	"log"
	"net/http"
    "math"
    "fmt"
	"time"
	"regexp"  // Paquete para expresiones regulares
	"strconv"
	"encoding/json"
	// "strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"		

	"api_compiladores/src/models"
	"api_compiladores/src/utils"
)

type ExampleUserCreate struct {
    Clave_Cliente string `json:"Clave_Cliente" example:"001"`
    Nombre        string `json:"Nombre" example:"Pedro"`
    Celular       string `json:"Celular" example:"9613214782"`
    Email         string `json:"Email" example:"correo@example.com"`
}

type ExampleUserPut struct {
    Nombre        string `json:"Nombre" example:"Pedro"`
    Celular       string `json:"Celular" example:"9613214782"`
    Email         string `json:"Email" example:"correo@example.com"`
}

var userCollection *mongo.Collection

var (
	// Expresion regular para validar que el campo Clave_Cliente sea un número entero
	identRegexNumeric = regexp.MustCompile(`^[0-9]*$`);
	// Expresion regular para validar que el campo Nombre sea una cadena de caracteres sin simbolos, numeros y 1 espacio entre palabras
	// identRegexNombre = regexp.MustCompile(`^[a-zA-ZáéíóúÁÉÍÓÚñÑüÜ\s]+$`)
	// // Expresion regular para validar que el campo Celular sea un número de 10 dígitos
	// identRegexCelular = regexp.MustCompile(`^(91[6-9]|93[24]|96[1-8]|99[24])\d{7}$`)
	// // Expresion regular para validar que el campo Email sea un correo electronico valido
	// identRegexEmail = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@(gmail\.com|hotmail\.com|yahoo\.com|outlook\.com|live\.com|icloud\.com|protonmail\.com|aol\.com|msn\.com|gmx\.com|ymail\.com|me\.com|mail\.com|zoho\.com|edu\.mx|edu\.com|edu\.org)$`)
)

// func ValidateUserInput(input *models.User) {
// 	errores := make(map[string][]string)

// 	// Validaciones para Nombre
// 	nombre := strings.TrimSpace(input.Nombre)
// 	if nombre == "" {
// 		errores["Nombre"] = append(errores["Nombre"], "El campo Nombre no puede estar vacío")
// 	} else {
// 		if !identRegexNombre.MatchString(nombre) {
// 			errores["Nombre"] = append(errores["Nombre"], "Solo se permiten letras y espacios")
// 		}
// 		if len(nombre) < 2 {
// 			errores["Nombre"] = append(errores["Nombre"], "El Nombre debe tener al menos 2 caracteres")
// 		}
// 		if len(nombre) > 50 {
// 			errores["Nombre"] = append(errores["Nombre"], "El Nombre no puede tener más de 50 caracteres")
// 		}
// 	}

// 	// Validaciones para Celular
// 	celular := strings.TrimSpace(input.Celular)
// 	if celular == "" {
// 		errores["Celular"] = append(errores["Celular"], "El campo Celular no puede estar vacío")
// 	} else {
// 		if len(celular) != 10 {
// 			errores["Celular"] = append(errores["Celular"], "El número de celular debe tener exactamente 10 dígitos")
// 		}
// 		if !identRegexCelular.MatchString(celular) {
// 			errores["Celular"] = append(errores["Celular"], "El celular debe tener una lada válida de Chiapas y el formato correcto")
// 		}
// 	}

// 	// Validaciones para Email
// 	email := strings.TrimSpace(input.Email)
// 	if email == "" {
// 		errores["Email"] = append(errores["Email"], "El campo Email no puede estar vacío")
// 	} else {
// 		if len(email) > 100 {
// 			errores["Email"] = append(errores["Email"], "El Email no puede exceder 100 caracteres")
// 		}
// 		if !identRegexEmail.MatchString(email) {
// 			errores["Email"] = append(errores["Email"], "El Email debe ser institucional o de proveedores conocidos (gmail, hotmail, yahoo, etc.)")
// 		}
// 	}

// 	// Asignar errores en el input
// 	if len(errores) > 0 {
// 		input.Errores = errores
// 	} else {
// 		input.Errores = nil
// 	}
// }

var exampleCreate = ExampleUserCreate{
        Clave_Cliente: "001",
        Nombre:        "Pedro",
        Celular:       "9613214782",
        Email:         "correo@example.com",
    }

var examplePut = ExampleUserPut{
        Nombre:        "Pedro",
        Celular:       "9613214782",
        Email:         "correo@example.com",
    }

func SetUserCollection(c *mongo.Collection) {
	userCollection = c
}

func CreateUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validación de campos obligatorios
	if user.Clave_Cliente == nil {
		message := "Error: Clave_Cliente es obligatorio"
		log.Println(message)

        jsonData := gin.H{
            "error":   message,
            "example": exampleCreate,
        }
		c.JSON(http.StatusBadRequest, jsonData)
		return
	}

	var Clave_Cliente string

    switch v := user.Clave_Cliente.(type) {
    case string:
        if !identRegexNumeric.MatchString(v) {
            message := "Error: " + user.Nombre + " Clave_Cliente no es un identificador válido"
            log.Println(message)
            jsonData := gin.H{
                "error":   message,
                "example": exampleCreate,
            }
            c.JSON(http.StatusBadRequest, jsonData)
            return
        }
		if len(v) > 10 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Clave de cliente no puede tener más de 10 caracteres"})
			return
		}

		// Convertir a entero
		intVal, err := strconv.Atoi(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Clave de cliente no es un número válido"})
			return
		}

		// rellenar con ceros a la izquierda
		newClave := fmt.Sprintf("%010d", intVal) // Rellenar con ceros a la izquierda
        Clave_Cliente = newClave

    case float64:
        if math.Mod(v, 1) != 0 {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Clave de cliente no puede ser decimal"})
            return
        }
        if v < 0 {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Clave de cliente no puede ser negativo"})
            return
        }
		if v > 9999999999 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Clave de cliente no puede ser mayor a 10 dígitos"})
			return
		}

        intVal := int(v)
		// rellenar con ceros a la izquierda
        newClave := fmt.Sprintf("%010d", intVal) 
        Clave_Cliente = newClave

    default:
        message := "Tipo de dato no válido para Clave_Cliente"
        log.Println(message)
        jsonData := gin.H{
            "error":   message,
            "example": exampleCreate,
        }
        c.JSON(http.StatusBadRequest, jsonData)
        return
    }

	// Crear nuevo ID
	user.ID = primitive.NewObjectID()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
    
	err := userCollection.FindOne(ctx, bson.M{"Clave_Cliente": Clave_Cliente}).Decode(&user)
	if err != nil {
		user.Clave_Cliente = Clave_Cliente
        _, err := userCollection.InsertOne(ctx, user)
        if err != nil {
            log.Println("Error al insertar usuario: ", err)
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al insertar usuario"})
            return
        }

        c.JSON(http.StatusCreated, user)
	} else {
		message := "Error: El usuario con Clave_Cliente " + Clave_Cliente + " ya existe"
        log.Println(message)
        c.JSON(http.StatusBadRequest, gin.H{"error": message})
        return
    }
}

func GetUsers(c *gin.Context) {
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

	// Asignamos el valor a page
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

	var users []models.User

	cacheKey := fmt.Sprintf("users_cache_page_%d", intPage)
	cachedUsers, err := utils.RedisClient.Get(utils.Ctx, cacheKey).Result()
	if err == nil {
		if err := json.Unmarshal([]byte(cachedUsers), &users); err == nil {
			// Si los datos están en caché, los devolvemos
			log.Println("Datos obtenidos de la caché")
			c.JSON(http.StatusOK, users)
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
	
	cursor, err := userCollection.Find(ctx, bson.M{}, options)
	if err != nil {
		log.Println("Error al obtener usuarios: ", err)
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

	log.Println("Datos obtenidos de la base de datos")
	jsonData, _ := json.Marshal(users)
	utils.RedisClient.Set(utils.Ctx, cacheKey, jsonData, 5*time.Minute)
	c.JSON(http.StatusOK, users)
}

func GetUser(c *gin.Context) {
	Clave_Cliente := c.Param("Clave_Cliente")

	var user models.User
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := userCollection.FindOne(ctx, bson.M{"Clave_Cliente": Clave_Cliente}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
		return
	}

	// ValidateUserInput(&user)

	c.JSON(http.StatusOK, user)
}

func UpdateUser(c *gin.Context) {
	Clave_Cliente := c.Param("Clave_Cliente")
	if Clave_Cliente == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El campo Clave_Cliente es obligatorio"})
		return
	}

	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validación de campos obligatorios
	if user.Nombre == "" {
		message := "Error: Nombre es obligatorio"
		log.Println(message)
		jsonData := gin.H{
			"error":   message,
			"example": examplePut,
		}
		c.JSON(http.StatusBadRequest, jsonData)
		return
	}

	if user.Celular == "" {
		message := "Error: Celular es obligatorio"
		log.Println(message)
		jsonData := gin.H{
			"error":   message,
			"example": examplePut,
		}
		c.JSON(http.StatusBadRequest, jsonData)
		return
	}

	if user.Email == "" {
		message := "Error: Email es obligatorio"
		log.Println(message)
		jsonData := gin.H{
			"error":   message,
			"example": examplePut,
		}
		c.JSON(http.StatusBadRequest, jsonData)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Solo actualizamos los campos válidos, no el ID
	update := bson.M{
		"$set": bson.M{
			"Nombre":        user.Nombre,
			"Celular":       user.Celular,
			"Email":         user.Email,
		},
	}

	result, err := userCollection.UpdateOne(ctx, bson.M{"Clave_Cliente": Clave_Cliente}, update)
	if err != nil {
		log.Println("Error al actualizar usuario: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar usuario"})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Usuario actualizado"})
}

func DeleteUser(c *gin.Context) {
	Clave_Cliente := c.Param("Clave_Cliente")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := userCollection.DeleteOne(ctx, bson.M{"Clave_Cliente": Clave_Cliente})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar usuario"})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Usuario eliminado"})
}
