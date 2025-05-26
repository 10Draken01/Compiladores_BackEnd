// controllers/user.controller.go
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

// Respuesta estructurada para APIs
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	Meta    *MetaInfo   `json:"meta,omitempty"`
}

type MetaInfo struct {
	Page       int    `json:"page,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Total      int64  `json:"total,omitempty"`
	CacheHit   bool   `json:"cache_hit,omitempty"`
	Source     string `json:"source,omitempty"`
	Timestamp  int64  `json:"timestamp"`
}

var (
	// Expresiones regulares reutilizables
	identRegexNumeric = regexp.MustCompile(`^[0-9]*$`)
	identRegexNombre  = regexp.MustCompile(`^[a-zA-ZáéíóúÁÉÍÓÚñÑüÜ\s]+$`)
	identRegexCelular = regexp.MustCompile(`^(91[6-9]|93[24]|96[1-8]|99[24])\d{7}$`)
	identRegexEmail   = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@(gmail\.com|hotmail\.com|yahoo\.com|outlook\.com|live\.com|icloud\.com|protonmail\.com|aol\.com|msn\.com|gmx\.com|ymail\.com|me\.com|mail\.com|zoho\.com|edu\.mx|edu\.com|edu\.org)$`)
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

// CreateCliente - Crear cliente con invalidación inteligente de caché
func CreateCliente(c *gin.Context) {
	var cliente models.Cliente
	if err := c.ShouldBindJSON(&cliente); err != nil {
		sendErrorResponse(c, http.StatusBadRequest, "Datos JSON inválidos", err.Error(), &exampleCreate)
		return
	}

	if cliente.Clave_Cliente == nil {
		sendErrorResponse(c, http.StatusBadRequest, "Clave_Cliente es obligatorio", nil, &exampleCreate)
		return
	}

	claveCliente, err := normalizeClaveCliente(cliente.Clave_Cliente)
	if err != nil {
		sendErrorResponse(c, http.StatusBadRequest, err.Error(), nil, &exampleCreate)
		return
	}

	cliente.ID = primitive.NewObjectID()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verificar existencia previa (primero en caché, luego en DB)
	if cachedCliente, found, _ := utils.GetCachedSingleCliente(claveCliente); found && cachedCliente != nil {
		log.Printf("Cliente %s ya existe (encontrado en caché)", claveCliente)
		sendErrorResponse(c, http.StatusBadRequest, 
			fmt.Sprintf("El cliente con Clave_Cliente %s ya existe", claveCliente), 
			nil, &exampleCreate)
		return
	}

	// Verificar en base de datos
	var existingCliente models.Cliente
	err = clienteCollection.FindOne(ctx, bson.M{"Clave_Cliente": claveCliente}).Decode(&existingCliente)
	if err == nil {
		// Cliente existe, guardarlo en caché para futuras consultas
		utils.CacheSingleCliente(claveCliente, existingCliente, utils.DefaultTTL)
		sendErrorResponse(c, http.StatusBadRequest,
			fmt.Sprintf("El cliente con Clave_Cliente %s ya existe", claveCliente),
			nil, &exampleCreate)
		return
	} else if err != mongo.ErrNoDocuments {
		log.Printf("Error al verificar existencia de cliente: %v", err)
		sendErrorResponse(c, http.StatusInternalServerError, "Error interno del servidor", nil, nil)
		return
	}

	cliente.Clave_Cliente = claveCliente
	utils.ValidateCliente(&cliente)

	// Insertar en base de datos
	if _, err := clienteCollection.InsertOne(ctx, cliente); err != nil {
		log.Printf("Error al insertar cliente: %v", err)
		sendErrorResponse(c, http.StatusInternalServerError, "Error al insertar cliente", nil, nil)
		return
	}

	// Invalidar caché relacionado (async para no bloquear la respuesta)
	go func() {
		if err := utils.InvalidateAllClientesCache(); err != nil {
			log.Printf("Error invalidando caché tras crear cliente: %v", err)
		}
		utils.UpdateCacheStats("invalidate")
	}()

	// Cachear el nuevo cliente
	go utils.CacheSingleCliente(claveCliente, cliente, utils.DefaultTTL)

	sendSuccessResponse(c, http.StatusCreated, "Cliente creado exitosamente", cliente, nil)
}

// GetClientes - Obtener clientes con caché optimizado
func GetClientes(c *gin.Context) {
	page := c.Param("page")

	if page == "" {
		sendErrorResponse(c, http.StatusBadRequest, 
			"El atributo page es obligatorio | debe ser un número entero del 1 al 10000", 
			nil, "1")
		return
	}

	if !identRegexNumeric.MatchString(page) {
		sendErrorResponse(c, http.StatusBadRequest,
			fmt.Sprintf("%s no es un identificador válido", page),
			nil, "1")
		return
	}

	intPage, err := strconv.Atoi(page)
	if err != nil {
		sendErrorResponse(c, http.StatusBadRequest, "Page debe ser un número válido", nil, "1")
		return
	}

	if intPage < 1 || intPage > 10000 {
		sendErrorResponse(c, http.StatusBadRequest,
			fmt.Sprintf("%s no es un número válido | debe ser un número entero del 1 al 10000", page),
			nil, "1")
		return
	}

	limit := 100
	skip := (intPage - 1) * limit

	// Intentar obtener desde caché primero
	if cachedClientes, found, err := utils.GetCachedClientesList(intPage); found && err == nil {
		utils.UpdateCacheStats("hit")
		
		meta := &MetaInfo{
			Page:      intPage,
			Limit:     limit,
			CacheHit:  true,
			Source:    "cache",
			Timestamp: time.Now().Unix(),
		}
		
		sendSuccessResponse(c, http.StatusOK, "Clientes obtenidos desde caché", cachedClientes, meta)
		return
	}

	// Si no está en caché, obtener de la base de datos
	utils.UpdateCacheStats("miss")
	
	var clientes []models.Cliente
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Contar total de documentos (solo si es la primera página para optimización)
	var total int64
	if intPage == 1 {
		if count, err := clienteCollection.CountDocuments(ctx, bson.M{}); err == nil {
			total = count
		}
	}

	options := options.Find()
	options.SetLimit(int64(limit))
	options.SetSkip(int64(skip))
	options.SetSort(bson.D{{"Clave_Cliente", 1}}) // Ordenar para consistencia

	cursor, err := clienteCollection.Find(ctx, bson.M{}, options)
	if err != nil {
		log.Printf("Error al obtener clientes: %v", err)
		sendErrorResponse(c, http.StatusInternalServerError, "Error al obtener clientes", nil, nil)
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var cliente models.Cliente
		if err := cursor.Decode(&cliente); err != nil {
			log.Printf("Error decodificando cliente: %v", err)
			continue
		}
		clientes = append(clientes, cliente)
	}

	// Guardar en caché de forma asíncrona
	go func() {
		if err := utils.CacheClientesList(intPage, clientes, utils.DefaultTTL); err != nil {
			log.Printf("Error guardando página %d en caché: %v", intPage, err)
		}
		utils.UpdateCacheStats("set")
	}()

	meta := &MetaInfo{
		Page:      intPage,
		Limit:     limit,
		Total:     total,
		CacheHit:  false,
		Source:    "database",
		Timestamp: time.Now().Unix(),
	}

	sendSuccessResponse(c, http.StatusOK, "Clientes obtenidos desde base de datos", clientes, meta)
}

// GetCliente - Obtener cliente individual con caché
func GetCliente(c *gin.Context) {
	claveCliente := c.Param("Clave_Cliente")
	
	// Intentar obtener desde caché primero
	if cachedCliente, found, err := utils.GetCachedSingleCliente(claveCliente); found && err == nil {
		utils.UpdateCacheStats("hit")
		
		meta := &MetaInfo{
			CacheHit:  true,
			Source:    "cache",
			Timestamp: time.Now().Unix(),
		}
		
		sendSuccessResponse(c, http.StatusOK, "Cliente obtenido desde caché", cachedCliente, meta)
		return
	}

	// Si no está en caché, obtener de la base de datos
	utils.UpdateCacheStats("miss")
	
	var cliente models.Cliente
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := clienteCollection.FindOne(ctx, bson.M{"Clave_Cliente": claveCliente}).Decode(&cliente)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			sendErrorResponse(c, http.StatusNotFound, "Cliente no encontrado", nil, nil)
		} else {
			log.Printf("Error obteniendo cliente %s: %v", claveCliente, err)
			sendErrorResponse(c, http.StatusInternalServerError, "Error interno del servidor", nil, nil)
		}
		return
	}

	// Guardar en caché de forma asíncrona
	go func() {
		if err := utils.CacheSingleCliente(claveCliente, cliente, utils.LongTTL); err != nil {
			log.Printf("Error guardando cliente %s en caché: %v", claveCliente, err)
		}
		utils.UpdateCacheStats("set")
	}()

	meta := &MetaInfo{
		CacheHit:  false,
		Source:    "database",
		Timestamp: time.Now().Unix(),
	}

	sendSuccessResponse(c, http.StatusOK, "Cliente obtenido desde base de datos", cliente, meta)
}

// UpdateCliente - Actualizar cliente con invalidación de caché
func UpdateCliente(c *gin.Context) {
	claveCliente := c.Param("Clave_Cliente")
	if claveCliente == "" {
		sendErrorResponse(c, http.StatusBadRequest, "El campo Clave_Cliente es obligatorio", nil, nil)
		return
	}

	var cliente models.Cliente
	if err := c.ShouldBindJSON(&cliente); err != nil {
		sendErrorResponse(c, http.StatusBadRequest, "Datos JSON inválidos", err.Error(), &examplePut)
		return
	}

	// mostrar propiedad errores en log
	log.Printf("Errores: %v", cliente.Errores)
	
	var clienteResponse models.Cliente
	clienteResponse.Clave_Cliente = claveCliente
	clienteResponse.Nombre = cliente.Nombre
	clienteResponse.Celular = cliente.Celular
	clienteResponse.Email = cliente.Email
	clienteResponse.Errores = cliente.Errores

	utils.ValidateCliente(&clienteResponse)

	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"Nombre":  clienteResponse.Nombre,
			"Celular": clienteResponse.Celular,
			"Email":   clienteResponse.Email,
			"Errores": clienteResponse.Errores,
		},
	}

	err := clienteCollection.FindOneAndUpdate(ctx, bson.M{"Clave_Cliente": claveCliente}, update).Decode(&cliente)
	if err != nil {
		log.Printf("Error al actualizar cliente %s: %v", claveCliente, err)
		sendErrorResponse(c, http.StatusInternalServerError, "Error al actualizar cliente", nil, nil)
		return
	}

	clienteResponse.ID = cliente.ID

	// Invalidar caché de forma asíncrona
	go func() {
		if err := utils.InvalidateClienteCache(claveCliente); err != nil {
			log.Printf("Error invalidando caché para cliente %s: %v", claveCliente, err)
		}
		utils.UpdateCacheStats("invalidate")
	}()

	// Cachear el cliente actualizado
	go utils.CacheSingleCliente(claveCliente, 
		clienteResponse, 
		utils.DefaultTTL)

	sendSuccessResponse(c, http.StatusOK, "Cliente actualizado exitosamente", clienteResponse, nil)
}

// DeleteCliente - Eliminar cliente con invalidación de caché
func DeleteCliente(c *gin.Context) {
	claveCliente := c.Param("Clave_Cliente")
	if claveCliente == "" {
		sendErrorResponse(c, http.StatusBadRequest, "El campo Clave_Cliente es obligatorio", nil, nil)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := clienteCollection.DeleteOne(ctx, bson.M{"Clave_Cliente": claveCliente})
	if err != nil {
		log.Printf("Error al eliminar cliente %s: %v", claveCliente, err)
		sendErrorResponse(c, http.StatusInternalServerError, "Error al eliminar cliente", nil, nil)
		return
	}

	if result.DeletedCount == 0 {
		sendErrorResponse(c, http.StatusNotFound, "Cliente no encontrado", nil, nil)
		return
	}

	// Invalidar caché de forma asíncrona
	go func() {
		if err := utils.InvalidateClienteCache(claveCliente); err != nil {
			log.Printf("Error invalidando caché para cliente eliminado %s: %v", claveCliente, err)
		}
		utils.UpdateCacheStats("invalidate")
	}()

	sendSuccessResponse(c, http.StatusOK, "Cliente eliminado exitosamente", nil, nil)
}

// GetCacheStats - Obtener estadísticas de caché
func GetCacheStats(c *gin.Context) {
	stats, err := utils.GetCacheStats()
	if err != nil {
		sendErrorResponse(c, http.StatusInternalServerError, "Error obteniendo estadísticas de caché", err.Error(), nil)
		return
	}

	meta := &MetaInfo{
		Source:    "redis",
		Timestamp: time.Now().Unix(),
	}

	sendSuccessResponse(c, http.StatusOK, "Estadísticas de caché obtenidas", stats, meta)
}

// ClearCache - Limpiar todo el caché (endpoint administrativo)
func ClearCache(c *gin.Context) {
	if err := utils.InvalidateAllClientesCache(); err != nil {
		sendErrorResponse(c, http.StatusInternalServerError, "Error limpiando caché", err.Error(), nil)
		return
	}

	utils.UpdateCacheStats("invalidate")
	sendSuccessResponse(c, http.StatusOK, "Caché limpiado exitosamente", nil, nil)
}

// HealthCheck - Verificar salud de Redis y MongoDB
func HealthCheck(c *gin.Context) {
	health := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"services":  make(map[string]interface{}),
	}

	// Verificar Redis
	if err := utils.CheckRedisHealth(); err != nil {
		health["services"].(map[string]interface{})["redis"] = map[string]interface{}{
			"status": "down",
			"error":  err.Error(),
		}
		health["status"] = "degraded"
	} else {
		health["services"].(map[string]interface{})["redis"] = map[string]interface{}{
			"status": "up",
		}
	}

	// Verificar MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := clienteCollection.Database().Client().Ping(ctx, nil); err != nil {
		health["services"].(map[string]interface{})["mongodb"] = map[string]interface{}{
			"status": "down",
			"error":  err.Error(),
		}
		health["status"] = "down"
	} else {
		health["services"].(map[string]interface{})["mongodb"] = map[string]interface{}{
			"status": "up",
		}
	}

	// Determinar código de respuesta HTTP
	statusCode := http.StatusOK
	if health["status"] == "down" {
		statusCode = http.StatusServiceUnavailable
	} else if health["status"] == "degraded" {
		statusCode = http.StatusPartialContent
	}

	c.JSON(statusCode, health)
}

// === FUNCIONES AUXILIARES ===

// normalizeClaveCliente - Normalizar y validar clave de cliente
func normalizeClaveCliente(claveCliente interface{}) (string, error) {
	var claveStr string
	
	switch v := claveCliente.(type) {
	case string:
		claveStr = v
	case float64:
		claveStr = fmt.Sprintf("%.0f", v)
	case int:
		claveStr = strconv.Itoa(v)
	case int64:
		claveStr = strconv.FormatInt(v, 10)
	default:
		return "", fmt.Errorf("tipo de Clave_Cliente no soportado")
	}

	// Validar que sea numérico
	if !identRegexNumeric.MatchString(claveStr) {
		return "", fmt.Errorf("Clave_Cliente debe contener solo números")
	}

	// Normalizar a formato de 3 dígitos con padding de ceros
	if intClave, err := strconv.Atoi(claveStr); err == nil {
		if intClave < 1 || intClave > 999 {
			return "", fmt.Errorf("Clave_Cliente debe estar entre 001 y 999")
		}
		return fmt.Sprintf("%03d", intClave), nil
	}

	return "", fmt.Errorf("Clave_Cliente inválida")
}

// sendSuccessResponse - Enviar respuesta exitosa estandarizada
func sendSuccessResponse(c *gin.Context, statusCode int, message string, data interface{}, meta *MetaInfo) {
	if meta == nil {
		meta = &MetaInfo{
			Timestamp: time.Now().Unix(),
		}
	} else if meta.Timestamp == 0 {
		meta.Timestamp = time.Now().Unix()
	}


	c.JSON(statusCode, data)
}

// sendErrorResponse - Enviar respuesta de error estandarizada
func sendErrorResponse(c *gin.Context, statusCode int, message string, errorData interface{}, example interface{}) {
	response := APIResponse{
		Success: false,
		Message: message,
		Error:   errorData,
		Meta: &MetaInfo{
			Timestamp: time.Now().Unix(),
		},
	}

	// Agregar ejemplo solo si se proporciona
	if example != nil {
		response.Data = map[string]interface{}{
			"example": example,
		}
	}

	c.JSON(statusCode, response)
}

// GetClientesCount - Obtener conteo total de clientes
func GetClientesCount(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	count, err := clienteCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Printf("Error obteniendo conteo de clientes: %v", err)
		sendErrorResponse(c, http.StatusInternalServerError, "Error obteniendo conteo", nil, nil)
		return
	}

	// Calcular páginas totales
	limit := 100
	totalPages := int(math.Ceil(float64(count) / float64(limit)))

	data := map[string]interface{}{
		"total_clientes": count,
		"total_pages":    totalPages,
		"items_per_page": limit,
	}

	meta := &MetaInfo{
		Total:     count,
		Source:    "database",
		Timestamp: time.Now().Unix(),
	}

	sendSuccessResponse(c, http.StatusOK, "Conteo obtenido exitosamente", data, meta)
}

// SearchClientes - Buscar clientes por criterios
func SearchClientes(c *gin.Context) {
	// Obtener parámetros de búsqueda
	nombre := c.Query("nombre")
	email := c.Query("email")
	celular := c.Query("celular")
	
	// Validar que al menos un criterio esté presente
	if nombre == "" && email == "" && celular == "" {
		sendErrorResponse(c, http.StatusBadRequest, 
			"Debe proporcionar al menos un criterio de búsqueda (nombre, email o celular)", 
			nil, 
			map[string]string{
				"ejemplo_url": "/api/clientes/search?nombre=Pedro&email=pedro@gmail.com",
			})
		return
	}

	// Construir filtro de búsqueda
	filter := bson.M{}
	if nombre != "" {
		// Búsqueda case-insensitive con regex
		filter["Nombre"] = primitive.Regex{
			Pattern: nombre,
			Options: "i",
		}
	}
	if email != "" {
		filter["Email"] = primitive.Regex{
			Pattern: email,
			Options: "i",
		}
	}
	if celular != "" {
		filter["Celular"] = celular
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Limitar resultados de búsqueda
	limit := int64(50)
	options := options.Find()
	options.SetLimit(limit)
	options.SetSort(bson.D{{"Clave_Cliente", 1}})

	cursor, err := clienteCollection.Find(ctx, filter, options)
	if err != nil {
		log.Printf("Error en búsqueda de clientes: %v", err)
		sendErrorResponse(c, http.StatusInternalServerError, "Error en búsqueda", nil, nil)
		return
	}
	defer cursor.Close(ctx)

	var clientes []models.Cliente
	for cursor.Next(ctx) {
		var cliente models.Cliente
		if err := cursor.Decode(&cliente); err != nil {
			log.Printf("Error decodificando cliente en búsqueda: %v", err)
			continue
		}
		clientes = append(clientes, cliente)
	}

	// Obtener conteo total de resultados
	totalCount, err := clienteCollection.CountDocuments(ctx, filter)
	if err != nil {
		log.Printf("Error obteniendo conteo de búsqueda: %v", err)
		totalCount = int64(len(clientes))
	}

	meta := &MetaInfo{
		Limit:     int(limit),
		Total:     totalCount,
		Source:    "database",
		Timestamp: time.Now().Unix(),
	}

	message := fmt.Sprintf("Búsqueda completada: %d resultados encontrados", len(clientes))
	sendSuccessResponse(c, http.StatusOK, message, clientes, meta)
}