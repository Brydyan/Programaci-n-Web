package controllers_test

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"
)

// versionedCache es un CacheService en memoria que SÍ registra los
// Increment, a diferencia de recordingCache (controllers_test.go), cuyo
// Increment devuelve 1 sin guardar nada.
//
// Sin eso no se puede observar el versionado de claves: la clave real lleva
// la versión embebida (`red:tree:v3:all`) y una comprobación necesita leer
// qué versión hay guardada, no sólo contar llamadas.
//
// ⚠️ Es thread-safe a propósito, y no por elegancia.
//
// El checkout lanza `actualizarContadores` en una goroutine post-COMMIT
// (checkoutController.go:88), que llama a Increment. Mientras tanto el test
// sondea `increments[...]` esperando el efecto. Go bloquea un mapa de forma
// INDIVIDUAL por operación: dos goroutines pueden no bloquearse y una
// escribir mientras la otra lee → "fatal error: concurrent map read and map
// write", que aborta el binario completo sin dejar tests en verde.
//
// El caché real (Redis) sí es concurrente; este doble debe serlo también, o
// el test midería un artefacto del doble y no del código de producción.
type versionedCache struct {
	mu sync.RWMutex

	data       map[string]string
	versions   map[string]int64
	gets       map[string]int
	sets       map[string]int
	deletes    map[string]int
	increments map[string]int
}

func newVersionedCache() *versionedCache {
	return &versionedCache{
		data:       map[string]string{},
		versions:   map[string]int64{},
		gets:       map[string]int{},
		sets:       map[string]int{},
		deletes:    map[string]int{},
		increments: map[string]int{},
	}
}

func (c *versionedCache) Get(_ context.Context, key string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.gets[key]++
	if v, ok := c.data[key]; ok {
		return v, nil
	}
	return "", errCacheMiss
}

func (c *versionedCache) Set(_ context.Context, key string, value interface{}, _ time.Duration) error {
	var raw string
	switch s := value.(type) {
	case string:
		raw = s
	case []byte:
		raw = string(s)
	default:
		b, err := json.Marshal(s)
		if err != nil {
			return err
		}
		raw = string(b)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.sets[key]++
	c.data[key] = raw
	return nil
}

func (c *versionedCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.deletes[key]++
	delete(c.data, key)
	return nil
}

// Increment imita a INCR de Redis: crea la clave si no existe y suma 1.
//
// Guardarlo también en `data` es lo importante: versionReportes/redVersion
// LEEN la versión con Get, no con Increment. Si Increment no dejara rastro en
// `data`, un Get posterior seguiría viendo versión 0 y toda clave versionada
// apuntaría a la misma — el test daría verde sobre un bug real.
func (c *versionedCache) Increment(_ context.Context, key string) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.increments[key]++
	c.versions[key]++
	v := c.versions[key]
	c.data[key] = strconv.FormatInt(v, 10)
	return v, nil
}

// accesos devuelve una copia de los contadores. Las copias evitan que quien
// inspecciona en el test compita por el mapa con la goroutine del checkout.
func (c *versionedCache) accesos() (gets, sets, deletes, increments map[string]int) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	gets = map[string]int{}
	sets = map[string]int{}
	deletes = map[string]int{}
	increments = map[string]int{}

	for k, v := range c.gets {
		gets[k] = v
	}
	for k, v := range c.sets {
		sets[k] = v
	}
	for k, v := range c.deletes {
		deletes[k] = v
	}
	for k, v := range c.increments {
		increments[k] = v
	}
	return
}

var errCacheMiss = errorString("cache miss")

type errorString string

func (e errorString) Error() string { return string(e) }
