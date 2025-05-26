// services/cache_service.go
package services

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "time"
    "crypto/md5"
    "encoding/hex"

    "api_compiladores/src/config"
    "api_compiladores/src/models"
)

type CacheService struct {
    defaultTTL    time.Duration
    longTTL       time.Duration
    shortTTL      time.Duration
    keyPrefix     string
}

func NewCacheService() *CacheService {
    return &CacheService{
        defaultTTL: 5 * time.Minute,
        longTTL:    30 * time.Minute,
        shortTTL:   1 * time.Minute,
        keyPrefix:  "api_compiladores:",
    }
}

// Generar hash MD5 para keys complejas
func (cs *CacheService) generateHash(data string) string {
    hash := md5.Sum([]byte(data))
    return hex.EncodeToString(hash[:])
}

// Generar clave de cache
func (cs *CacheService) generateKey(keyType, identifier string) string {
    return fmt.Sprintf("%s%s:%s", cs.keyPrefix, keyType, identifier)
}

// Cache para clientes paginados
func (cs *CacheService) SetClientesPage(page int, clientes []models.Cliente) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    key := cs.generateKey("clientes_page", fmt.Sprintf("%d", page))
    
    // Serializar con compresión básica
    data, err := json.Marshal(clientes)
    if err != nil {
        return fmt.Errorf("error marshaling clientes: %v", err)
    }

    // Pipeline para operaciones atómicas
    pipe := config.RedisClient.Pipeline()
    pipe.Set(ctx, key, data, cs.defaultTTL)
    pipe.Set(ctx, key+"_meta", map[string]interface{}{
        "count":      len(clientes),
        "cached_at":  time.Now().Unix(),
        "page":       page,
    }, cs.defaultTTL)
    
    _, err = pipe.Exec(ctx)
    if err != nil {
        log.Printf("Error caching clientes page %d: %v", page, err)
        return err
    }

    log.Printf("Cache: Página %d almacenada con %d clientes", page, len(clientes))
    return nil
}

func (cs *CacheService) GetClientesPage(page int) ([]models.Cliente, bool) {
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    key := cs.generateKey("clientes_page", fmt.Sprintf("%d", page))
    
    data, err := config.RedisClient.Get(ctx, key).Result()
    if err != nil {
        if err.Error() != "redis: nil" {
            log.Printf("Error getting cached page %d: %v", page, err)
        }
        return nil, false
    }

    var clientes []models.Cliente
    if err := json.Unmarshal([]byte(data), &clientes); err != nil {
        log.Printf("Error unmarshaling cached clientes: %v", err)
        // Limpiar cache corrupto
        cs.InvalidateClientesPage(page)
        return nil, false
    }

    log.Printf("Cache: Página %d recuperada con %d clientes", page, len(clientes))
    return clientes, true
}

// Cache para cliente individual
func (cs *CacheService) SetCliente(claveCliente string, cliente models.Cliente) error {
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    key := cs.generateKey("cliente", claveCliente)
    
    data, err := json.Marshal(cliente)
    if err != nil {
        return fmt.Errorf("error marshaling cliente: %v", err)
    }

    err = config.RedisClient.Set(ctx, key, data, cs.longTTL).Err()
    if err != nil {
        log.Printf("Error caching cliente %s: %v", claveCliente, err)
        return err
    }

    return nil
}

func (cs *CacheService) GetCliente(claveCliente string) (models.Cliente, bool) {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    key := cs.generateKey("cliente", claveCliente)
    
    data, err := config.RedisClient.Get(ctx, key).Result()
    if err != nil {
        return models.Cliente{}, false
    }

    var cliente models.Cliente
    if err := json.Unmarshal([]byte(data), &cliente); err != nil {
        log.Printf("Error unmarshaling cached cliente: %v", err)
        cs.InvalidateCliente(claveCliente)
        return models.Cliente{}, false
    }

    return cliente, true
}

// Invalidación de cache
func (cs *CacheService) InvalidateClientesPage(page int) error {
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)  
    defer cancel()

    key := cs.generateKey("clientes_page", fmt.Sprintf("%d", page))
    
    pipe := config.RedisClient.Pipeline()
    pipe.Del(ctx, key)
    pipe.Del(ctx, key+"_meta")
    
    _, err := pipe.Exec(ctx)
    return err
}

func (cs *CacheService) InvalidateCliente(claveCliente string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    key := cs.generateKey("cliente", claveCliente)
    return config.RedisClient.Del(ctx, key).Err()
}

// Invalidación masiva cuando se crean/actualizan/eliminan clientes
func (cs *CacheService) InvalidateAllClientesPages() error {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    pattern := cs.generateKey("clientes_page", "*")
    
    keys, err := config.RedisClient.Keys(ctx, pattern).Result()
    if err != nil {
        return err
    }

    if len(keys) > 0 {
        // Usar pipeline para eliminar todas las keys de una vez
        pipe := config.RedisClient.Pipeline()
        for _, key := range keys {
            pipe.Del(ctx, key)
        }
        
        _, err = pipe.Exec(ctx)
        if err != nil {
            return err
        }
        
        log.Printf("Cache: Invalidadas %d páginas de clientes", len(keys))
    }

    return nil
}

// Cache para estadísticas y conteos
func (cs *CacheService) SetTotalClientes(count int64) error {
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    key := cs.generateKey("stats", "total_clientes")
    
    data := map[string]interface{}{
        "count":     count,
        "cached_at": time.Now().Unix(),
    }
    
    return config.RedisClient.HMSet(ctx, key, data).Err()
}

func (cs *CacheService) GetTotalClientes() (int64, bool) {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    key := cs.generateKey("stats", "total_clientes")
    
    result, err := config.RedisClient.HGet(ctx, key, "count").Result()
    if err != nil {
        return 0, false
    }

    var count int64
    if err := json.Unmarshal([]byte(result), &count); err == nil {
        return count, true
    }

    return 0, false
}

// Cache distribuido con locks para evitar cache stampede
func (cs *CacheService) GetOrSetWithLock(key string, ttl time.Duration, fetchFunc func() (interface{}, error)) (interface{}, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    lockKey := key + ":lock"
    
    // Intentar obtener del cache primero
    cached, err := config.RedisClient.Get(ctx, key).Result()
    if err == nil {
        var result interface{}
        if err := json.Unmarshal([]byte(cached), &result); err == nil {
            return result, nil
        }
    }

    // Intentar adquirir lock
    acquired, err := config.RedisClient.SetNX(ctx, lockKey, "locked", 30*time.Second).Result()
    if err != nil {
        return nil, fmt.Errorf("error adquiriendo lock: %v", err)
    }

    if !acquired {
        // Esperar un momento y reintentar obtener del cache
        time.Sleep(100 * time.Millisecond)
        cached, err := config.RedisClient.Get(ctx, key).Result()
        if err == nil {
            var result interface{}
            if err := json.Unmarshal([]byte(cached), &result); err == nil {
                return result, nil
            }
        }
        return nil, fmt.Errorf("unable to acquire lock and no cached data available")
    }

    // Tenemos el lock, obtener datos
    defer config.RedisClient.Del(ctx, lockKey)

    data, err := fetchFunc()
    if err != nil {
        return nil, err
    }

    // Guardar en cache
    serialized, err := json.Marshal(data)
    if err != nil {
        return data, nil // Retornar datos aunque no se puedan cachear
    }

    config.RedisClient.Set(ctx, key, serialized, ttl)
    return data, nil
}

// Utilidades de debugging y monitoreo
func (cs *CacheService) GetCacheStats() map[string]interface{} {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    stats := make(map[string]interface{})
    
    // Obtener info general de Redis
    info, err := config.RedisClient.Info(ctx, "memory").Result()
    if err == nil {
        stats["redis_info"] = info
    }

    // Contar keys por tipo
    patterns := []string{
        cs.keyPrefix + "clientes_page:*",
        cs.keyPrefix + "cliente:*",
        cs.keyPrefix + "stats:*",
    }

    for _, pattern := range patterns {
        keys, err := config.RedisClient.Keys(ctx, pattern).Result()
        if err == nil {
            stats[pattern] = len(keys)
        }
    }

    return stats
}

func (cs *CacheService) FlushAll() error {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    pattern := cs.keyPrefix + "*"
    keys, err := config.RedisClient.Keys(ctx, pattern).Result() 
    if err != nil {
        return err
    }

    if len(keys) > 0 {
        return config.RedisClient.Del(ctx, keys...).Err()
    }

    return nil
}