package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"

	"multicatalogo-backend/domain"
	"multicatalogo-backend/interfaces"
	"multicatalogo-backend/middleware"
)

type NetworkController struct {
	repo  interfaces.DBRepository
	cache interfaces.CacheService
}

func NewNetworkController(repo interfaces.DBRepository, cache interfaces.CacheService) *NetworkController {
	return &NetworkController{repo: repo, cache: cache}
}

// ── Caché versionada de la red ──────────────────────────────────────────────
//
// Invalidar la red "a pelo" (DEL de cada clave de usuario) es imposible de
// hacer bien: un checkout acredita hasta 3 nodos del upline y un registro
// cuelga un nodo del padre. Ningún controller sabe, en general, qué claves
// quedaron obsoletas.
//
// Solución: versionado. La clave real lleva dentro un contador global
// (`red:ver`). Cuando algo escribe en la red se incrementa el contador; las
// claves viejas dejan de direccionarse de inmediato y expiran solas por TTL.
// No hace falta enumerar ni purgar nada.

const (
	redVersionKey = "red:ver"
	redTTL        = 10 * time.Minute
	redScopeAll   = "all"
)

// redVersion lee el contador de versión. Si Redis está caído devuelve 0: la
// caché no funciona de todos modos, así que la API responde siempre desde
// PostgreSQL (degradación silenciosa, sin error al cliente).
func (nc *NetworkController) redVersion(ctx context.Context) int64 {
	if nc.cache == nil {
		return 0
	}
	raw, err := nc.cache.Get(ctx, redVersionKey)
	if err != nil {
		return 0
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// redCacheKey devuelve la clave del árbol para un ámbito dado, con la versión
// actual embebida.
func (nc *NetworkController) redCacheKey(ctx context.Context, scope string) string {
	return fmt.Sprintf("red:tree:v%d:%s", nc.redVersion(ctx), scope)
}

// GetRed devuelve el árbol global. Admin only (ver routes.go): la estructura
// completa de la red y sus ventas no es información pública.
//
// El JSON es un array de nodos anidados (components.schemas.NodoRed).
func (nc *NetworkController) GetRed(c *fiber.Ctx) error {
	return nc.responderArbol(c, redScopeAll, func() ([]domain.NodoPlano, error) {
		return nc.repo.GetRedGlobal()
	})
}

// GetRedMia devuelve el subárbol del usuario del token (claim `sub`), no el
// árbol completo. Cada usuario ve su red: la información accionable para un
// distribuidor es "quién depende de mí", no la de los demás.
func (nc *NetworkController) GetRedMia(c *fiber.Ctx) error {
	claimsAny := c.Locals(middleware.LocalUserKey)
	claims, ok := claimsAny.(middleware.JWTClaims)
	usuarioID := claims.Sub
	if !ok {
		return middleware.RespondError(c,
			fiber.StatusUnauthorized,
			string(middleware.CodeUnauthorized),
			"token sin identidad de usuario",
		)
	}
	scope := fmt.Sprintf("u%d", usuarioID)
	return nc.responderArbol(c, scope, func() ([]domain.NodoPlano, error) {
		return nc.repo.GetRedPropia(usuarioID)
	})
}

// Invalidación: subir `redVersionKey` con cache.Increment DESPUÉS de una
// escritura que cambie la red. Lo hacen directamente los controllers que
// escriben (AuthController.Register y CheckoutController), porque ninguno tiene
// una referencia al NetworkController — que vive en la capa de lectura.
//
// No hay un método InvalidateRed() para llamar desde fuera: añadirlo
// invitaría a usarlo desde un sitio donde el cache inyectado es otro, y a que
// la invalidación se application sólo a veces. El mecanismo es uno solo:
// Increment(redVersionKey). Ver Constantes arriba.

// responderArbol es el camino común de las dos vistas: caché-aside sobre el
// árbol ya ensamblado (el JSON es lo que se cachea, no los nodos sueltos).
func (nc *NetworkController) responderArbol(
	c *fiber.Ctx,
	scope string,
	cargar func() ([]domain.NodoPlano, error),
) error {
	ctx := context.Background()
	key := nc.redCacheKey(ctx, scope)

	if nc.cache != nil {
		if cached, err := nc.cache.Get(ctx, key); err == nil {
			var arbol []domain.NodoRed
			if jsonErr := json.Unmarshal([]byte(cached), &arbol); jsonErr == nil {
				return c.Status(fiber.StatusOK).JSON(arbol)
			}
		} else if err != redis.Nil {
			// Redis caído: seguimos a PostgreSQL sin quejarnos al cliente.
			_ = err
		}
	}

	nodos, err := cargar()
	if err != nil {
		return middleware.RespondError(c,
			fiber.StatusInternalServerError,
			string(middleware.CodeInternal),
			"error consultando la red",
			err.Error(),
		)
	}

	arbol := domain.ConstruirArbol(nodos)
	if nc.cache != nil {
		if data, jsonErr := json.Marshal(arbol); jsonErr == nil {
			//nolint:errcheck // best-effort: si falla, sólo se pierde caché
			nc.cache.Set(ctx, key, data, redTTL)
		}
	}
	return c.Status(fiber.StatusOK).JSON(arbol)
}
