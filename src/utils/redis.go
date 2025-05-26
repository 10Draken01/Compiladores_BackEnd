// utils/redis.go
package utils

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "strconv"
    "time"

    "github.com/go-redis/redis/v8"
    "api_compiladores/src/models"
)

var (
    Ctx         = context.Background()
    RedisClient *redis.Client
)

// Constantes para configuración de caché
const (
    DefaultTTL           = 5 * time.Minute
    LongTTL             = 30 * time.Minute
    ShortTTL            = 1 * time.Minute
    MaxRetries          = 3
    ClientesCachePrefix = "clientes:"
    SingleClientePrefix = "cliente:"
    StatsPrefix         = "stats:"
)

// Configuración de Redis
type RedisConfig struct {
    Addr         string
    Password     string
    DB           int
    PoolSize     int
    MinIdleConns int
    MaxRetries   int
    DialTimeout  time.Duration
    ReadTimeout  time.Duration
    WriteTimeout time.Duration
}

// Inicializar Redis con configuración mejorada
func ConnectRedis() {
    config := getRedisConfig()
    
    RedisClient = redis.NewClient(&redis.Options{
        Addr:         config.Addr,
        Password:     config.Password,
        DB:           config.DB,
        PoolSize:     config.PoolSize,
        MinIdleConns: config.MinIdleConns,
        MaxRetries:   config.MaxRetries,
        DialTimeout:  config.DialTimeout,
        ReadTimeout:  config.ReadTimeout,
        WriteTimeout: config.WriteTimeout,
    })

    // Verificar conexión
    if err := RedisClient.Ping(Ctx).Err(); err != nil {
        log.Printf("Error conectando a Redis: %v", err)
        log.Println("Continuando sin caché Redis...")
        RedisClient = nil
        return
    }

    log.Println("✅ Conexión a Redis establecida correctamente")
    
    // Configurar hooks para logging (opcional)
    RedisClient.AddHook(&LoggingHook{})
}

// Obtener configuración de Redis desde variables de entorno
func getRedisConfig() RedisConfig {
    return RedisConfig{
        Addr:         getEnvWithDefault("REDIS_ADDR", "localhost:6379"),
        Password:     getEnvWithDefault("REDIS_PASSWORD", ""),
        DB:           getIntEnvWithDefault("REDIS_DB", 0),
        PoolSize:     getIntEnvWithDefault("REDIS_POOL_SIZE", 100),
        MinIdleConns: getIntEnvWithDefault("REDIS_MIN_IDLE_CONNS", 10),
        MaxRetries:   getIntEnvWithDefault("REDIS_MAX_RETRIES", 3),
        DialTimeout:  getDurationEnvWithDefault("REDIS_DIAL_TIMEOUT", 5*time.Second),
        ReadTimeout:  getDurationEnvWithDefault("REDIS_READ_TIMEOUT", 3*time.Second),
        WriteTimeout: getDurationEnvWithDefault("REDIS_WRITE_TIMEOUT", 3*time.Second),
    }
}

// Funciones auxiliares para leer variables de entorno
func getEnvWithDefault(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func getIntEnvWithDefault(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if intValue, err := strconv.Atoi(value); err == nil {
            return intValue
        }
    }
    return defaultValue
}

func getDurationEnvWithDefault(key string, defaultValue time.Duration) time.Duration {
    if value := os.Getenv(key); value != "" {
        if duration, err := time.ParseDuration(value); err == nil {
            return duration
        }
    }
    return defaultValue
}

// === OPERACIONES DE CACHÉ PARA CLIENTES ===

// Obtener lista de clientes desde caché
func GetCachedClientesList(page int) ([]models.Cliente, bool, error) {
    if RedisClient == nil {
        return nil, false, fmt.Errorf("Redis no disponible")
    }

    key := fmt.Sprintf("%spage_%d", ClientesCachePrefix, page)
    
    cachedData, err := RedisClient.Get(Ctx, key).Result()
    if err == redis.Nil {
        return nil, false, nil // No encontrado en caché
    }
    if err != nil {
        return nil, false, fmt.Errorf("error obteniendo de caché: %w", err)
    }

    var clientes []models.Cliente
    if err := json.Unmarshal([]byte(cachedData), &clientes); err != nil {
        // Si hay error deserializando, limpiar la entrada corrupta
        RedisClient.Del(Ctx, key, key+"_meta")
        return nil, false, fmt.Errorf("error deserializando caché: %w", err)
    }

    log.Printf("🎯 Página %d obtenida de caché con %d clientes", page, len(clientes))
    return clientes, true, nil
}

// Guardar cliente individual en caché
func CacheSingleCliente(claveCliente string, cliente models.Cliente, ttl time.Duration) error {
    if RedisClient == nil {
        return fmt.Errorf("Redis no disponible")
    }

    key := fmt.Sprintf("%s%s", SingleClientePrefix, claveCliente)
    
    jsonData, err := json.Marshal(cliente)
    if err != nil {
        return fmt.Errorf("error serializando cliente: %w", err)
    }

    err = RedisClient.Set(Ctx, key, jsonData, ttl).Err()
    if err != nil {
        return fmt.Errorf("error guardando cliente en caché: %w", err)
    }

    log.Printf("📦 Cliente %s cacheado", claveCliente)
    return nil
}

// Obtener cliente individual desde caché
func GetCachedSingleCliente(claveCliente string) (*models.Cliente, bool, error) {
    if RedisClient == nil {
        return nil, false, fmt.Errorf("Redis no disponible")
    }

    key := fmt.Sprintf("%s%s", SingleClientePrefix, claveCliente)
    
    cachedData, err := RedisClient.Get(Ctx, key).Result()
    if err == redis.Nil {
        return nil, false, nil // No encontrado
    }
    if err != nil {
        return nil, false, fmt.Errorf("error obteniendo cliente de caché: %w", err)
    }

    var cliente models.Cliente
    if err := json.Unmarshal([]byte(cachedData), &cliente); err != nil {
        RedisClient.Del(Ctx, key)
        return nil, false, fmt.Errorf("error deserializando cliente: %w", err)
    }

    log.Printf("🎯 Cliente %s obtenido de caché", claveCliente)
    return &cliente, true, nil
}

// === INVALIDACIÓN DE CACHÉ ===

// Invalidar caché de un cliente específico
func InvalidateClienteCache(claveCliente string) error {
    if RedisClient == nil {
        return nil
    }

    // Usar pipeline para eliminar múltiples keys relacionadas
    pipe := RedisClient.Pipeline()
    
    // Eliminar cliente individual
    clienteKey := fmt.Sprintf("%s%s", SingleClientePrefix, claveCliente)
    pipe.Del(Ctx, clienteKey)
    
    // Eliminar todas las páginas de la lista de clientes
    // Esto podría optimizarse usando un set para trackear páginas activas
    for i := 1; i <= 100; i++ { // Asumiendo máximo 100 páginas
        pageKey := fmt.Sprintf("%spage_%d", ClientesCachePrefix, i)
        pipe.Del(Ctx, pageKey, pageKey+"_meta")
    }

    _, err := pipe.Exec(Ctx)
    if err != nil {
        return fmt.Errorf("error invalidando caché: %w", err)
    }

    log.Printf("🗑️ Caché invalidado para cliente %s", claveCliente)
    return nil
}

// Invalidar todo el caché de clientes
func InvalidateAllClientesCache() error {
    if RedisClient == nil {
        return nil
    }

    // Usar SCAN para encontrar todas las keys con nuestros prefijos
    patterns := []string{
        ClientesCachePrefix + "*",
        SingleClientePrefix + "*",
    }

    for _, pattern := range patterns {
        keys, err := RedisClient.Keys(Ctx, pattern).Result()
        if err != nil {
            log.Printf("Error obteniendo keys con patrón %s: %v", pattern, err)
            continue
        }

        if len(keys) > 0 {
            _, err = RedisClient.Del(Ctx, keys...).Result()
            if err != nil {
                log.Printf("Error eliminando keys: %v", err)
            } else {
                log.Printf("🗑️ Eliminadas %d keys con patrón %s", len(keys), pattern)
            }
        }
    }

    return nil
}

// === ESTADÍSTICAS Y MONITOREO ===

// Guardar estadísticas de uso
func UpdateCacheStats(operation string) {
    if RedisClient == nil {
        return
    }

    pipe := RedisClient.Pipeline()
    today := time.Now().Format("2006-01-02")
    
    // Contadores diarios
    pipe.Incr(Ctx, fmt.Sprintf("%s%s:%s", StatsPrefix, operation, today))
    pipe.Expire(Ctx, fmt.Sprintf("%s%s:%s", StatsPrefix, operation, today), 7*24*time.Hour)
    
    // Contador total
    pipe.Incr(Ctx, fmt.Sprintf("%s%s:total", StatsPrefix, operation))
    
    pipe.Exec(Ctx)
}

// Obtener estadísticas de caché
func GetCacheStats() (map[string]interface{}, error) {
    if RedisClient == nil {
        return nil, fmt.Errorf("Redis no disponible")
    }

    stats := make(map[string]interface{})
    today := time.Now().Format("2006-01-02")

    // Obtener estadísticas básicas
    operations := []string{"hit", "miss", "set", "invalidate"}
    
    for _, op := range operations {
        dailyKey := fmt.Sprintf("%s%s:%s", StatsPrefix, op, today)
        totalKey := fmt.Sprintf("%s%s:total", StatsPrefix, op)
        
        dailyCount, _ := RedisClient.Get(Ctx, dailyKey).Int()
        totalCount, _ := RedisClient.Get(Ctx, totalKey).Int()
        
        stats[op] = map[string]int{
            "today": dailyCount,
            "total": totalCount,
        }
    }

    // Información de memoria de Redis
    info, err := RedisClient.Info(Ctx, "memory").Result()
    if err == nil {
        stats["memory_info"] = info
    }

    return stats, nil
}

// === UTILIDADES ADICIONALES ===

// Verificar salud de Redis
func CheckRedisHealth() error {
    if RedisClient == nil {
        return fmt.Errorf("Redis no inicializado")
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    return RedisClient.Ping(ctx).Err()
}

// Limpiar caché expirado manualmente (útil para mantenimiento)
func CleanExpiredCache() error {
    if RedisClient == nil {
        return fmt.Errorf("Redis no disponible")
    }

    // Redis maneja esto automáticamente, pero podemos forzar una limpieza
    patterns := []string{
        ClientesCachePrefix + "*",
        SingleClientePrefix + "*",
    }

    for _, pattern := range patterns {
        keys, err := RedisClient.Keys(Ctx, pattern).Result()
        if err != nil {
            continue
        }

        for _, key := range keys {
            ttl := RedisClient.TTL(Ctx, key).Val()
            if ttl == -1 { // Sin expiración
                RedisClient.Expire(Ctx, key, DefaultTTL)
            }
        }
    }

    return nil
}

// Hook para logging de operaciones Redis (opcional)
type LoggingHook struct{}

func (h *LoggingHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
    return ctx, nil
}

func (h *LoggingHook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
    if cmd.Err() != nil && cmd.Err() != redis.Nil {
        log.Printf("🔴 Redis comando falló: %s - Error: %v", cmd.String(), cmd.Err())
    }
    return nil
}

func (h *LoggingHook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
    return ctx, nil
}

func (h *LoggingHook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
    for _, cmd := range cmds {
        if cmd.Err() != nil && cmd.Err() != redis.Nil {
            log.Printf("🔴 Redis pipeline falló: %s - Error: %v", cmd.String(), cmd.Err())
        }
    }
    return nil
}
// Guardar lista de clientes en caché con compresión
func CacheClientesList(page int, clientes []models.Cliente, ttl time.Duration) error {
    if RedisClient == nil {
        return fmt.Errorf("Redis no disponible")
    }

    key := fmt.Sprintf("%spage_%d", ClientesCachePrefix, page)
    
    // Serializar a JSON
    jsonData, err := json.Marshal(clientes)
    if err != nil {
        return fmt.Errorf("error serializando clientes: %w", err)
    }

    // Crear metadata y serializarla también
    metadata := map[string]interface{}{
        "count":      len(clientes),
        "cached_at":  time.Now().Unix(),
        "page":       page,
    }
    
    metaJsonData, err := json.Marshal(metadata)
    if err != nil {
        return fmt.Errorf("error serializando metadata: %w", err)
    }

    // Usar pipeline para operaciones múltiples
    pipe := RedisClient.Pipeline()
    pipe.Set(Ctx, key, jsonData, ttl)
    pipe.Set(Ctx, key+"_meta", metaJsonData, ttl)

    _, err = pipe.Exec(Ctx)
    if err != nil {
        return fmt.Errorf("error guardando en caché: %w", err)
    }

    log.Printf("📦 Página %d cacheada con %d clientes", page, len(clientes))
    return nil
}

// También necesitas actualizar la función que lee los metadatos si los usas
func GetCachedClientesListWithMeta(page int) ([]models.Cliente, map[string]interface{}, bool, error) {
    if RedisClient == nil {
        return nil, nil, false, fmt.Errorf("Redis no disponible")
    }

    key := fmt.Sprintf("%spage_%d", ClientesCachePrefix, page)
    metaKey := key + "_meta"
    
    // Usar pipeline para obtener ambos valores
    pipe := RedisClient.Pipeline()
    dataCmd := pipe.Get(Ctx, key)
    metaCmd := pipe.Get(Ctx, metaKey)
    
    _, err := pipe.Exec(Ctx)
    if err != nil {
        return nil, nil, false, fmt.Errorf("error obteniendo de caché: %w", err)
    }

    // Verificar si los datos existen
    cachedData, err := dataCmd.Result()
    if err == redis.Nil {
        return nil, nil, false, nil // No encontrado en caché
    }
    if err != nil {
        return nil, nil, false, fmt.Errorf("error obteniendo datos: %w", err)
    }

    // Deserializar clientes
    var clientes []models.Cliente
    if err := json.Unmarshal([]byte(cachedData), &clientes); err != nil {
        // Si hay error deserializando, limpiar las entradas corruptas
        RedisClient.Del(Ctx, key, metaKey)
        return nil, nil, false, fmt.Errorf("error deserializando clientes: %w", err)
    }

    // Deserializar metadata
    var metadata map[string]interface{}
    metaData, err := metaCmd.Result()
    if err == nil {
        if err := json.Unmarshal([]byte(metaData), &metadata); err != nil {
            log.Printf("Error deserializando metadata: %v", err)
            metadata = map[string]interface{}{
                "count": len(clientes),
                "page":  page,
            }
        }
    } else {
        metadata = map[string]interface{}{
            "count": len(clientes),
            "page":  page,
        }
    }

    log.Printf("🎯 Página %d obtenida de caché con %d clientes", page, len(clientes))
    return clientes, metadata, true, nil
}