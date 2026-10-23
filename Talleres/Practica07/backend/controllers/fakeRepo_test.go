package controllers_test

import (
	"errors"

	"multicatalogo-backend/domain"
	"multicatalogo-backend/interfaces"
	"multicatalogo-backend/internal/dto"
	"multicatalogo-backend/models"
)

// fakeRepo implementa interfaces.DBRepository sin tocar PostgreSQL.
//
// Existe porque la mayoría de la lógica de los controllers NO está en el SQL:
// está en el mapeo de query-params, en la traducción de errores a códigos HTTP
// y en el armored de caché. Probar eso contra la DB real obliga a levantar
// Postgres para cada caso, y en esta máquina no siempre hay uno — la suite se
// saltaba en silencio y el paquete se quedaba en 14.9% de cobertura.
//
// Uso: se declara sólo lo que el test necesita y el resto devuelve ceros.
//
//	repo := &fakeRepo{
//	    GetTopProductosFn: func(limit int) ([]models.TopProducto, error) {
//	        return fixtures, nil
//	    },
//	}
//	app := appConRepo(repo, &MockCacheService{})
//
// Si un método no se sobreescribe, devuelve un valor cero válido: el test que
// no lo usa no se rompe por un nil inesperado.
type fakeRepo struct {
	LoginFn                   func(email, password string) (*models.Usuario, error)
	RegistrarFn               func(req models.RegisterRequest) (*models.RegisterResult, error)
	GetUsuarioByCodigoRefFn   func(codigo string) (*models.Referido, error)
	GetReferidoIDByUsuarioFn  func(usuarioID int) (*int, error)
	GetAllProductosFn         func() ([]models.Producto, error)
	GetProductoByIDFn         func(id int) (*models.Producto, error)
	GetProductosAdminFn       func(filtro interfaces.ProductoFiltro) ([]models.Producto, int, error)
	CreateProductoFn          func(d dto.ProductoCreate) (*models.Producto, error)
	UpdateProductoFn          func(id int, d dto.ProductoUpdate) (*models.Producto, error)
	SoftDeleteProductoFn      func(id int) error
	GetRedGlobalFn            func() ([]domain.NodoPlano, error)
	GetRedPropiaFn            func(usuarioID int) ([]domain.NodoPlano, error)
	CreatePedidoConUsuarioFn  func(req dto.CheckoutRequest, usuarioID int) (*models.PedidoDetalle, error)
	GetMisPedidosFn           func(usuarioID, page, limit int) ([]models.PedidoResumen, int, error)
	GetPedidosAdminFn         func(filtro interfaces.PedidoFiltro) ([]models.PedidoResumen, int, error)
	GetPedidoDetalleFn        func(id int) (*models.PedidoDetalle, error)
	CambiarEstadoPedidoFn     func(id int, nuevoEstado domain.EstadoPedido) error
	GetComisionesFn           func(filtro interfaces.ComisionFiltro) ([]models.Comision, int, error)
	GetTotalesComisionesFn    func(referidoID *int) ([]models.ComisionTotal, error)
	ActualizarVentasUplineFn  func(emailUsuario string, total float64) error
	LiquidarFn                func(referidoID int) (*models.Liquidacion, error)
	GetLiquidacionesFn        func(page, limit int) ([]models.Liquidacion, int, error)
	GetLiquidacionesDeFn      func(referidoID, page, limit int) ([]models.Liquidacion, int, error)
	GetDeudaFn                func() ([]models.DeudaDistribuidor, error)
	GetReporteResumenFn       func(filtro interfaces.ReporteFiltro) (*models.ReporteResumen, error)
	GetTopProductosFn         func(limit int) ([]models.TopProducto, error)
	GetTopReferidosFn         func(limit int) ([]models.TopReferido, error)
	GetComisionesPendientesFn func(limit int) ([]models.Comision, error)
}

var _ interfaces.DBRepository = (*fakeRepo)(nil)

func (f *fakeRepo) Login(email, password string) (*models.Usuario, error) {
	if f.LoginFn != nil {
		return f.LoginFn(email, password)
	}
	return nil, nil
}

func (f *fakeRepo) Registrar(req models.RegisterRequest) (*models.RegisterResult, error) {
	if f.RegistrarFn != nil {
		return f.RegistrarFn(req)
	}
	// Nunca (nil, nil): el controller deserializa el resultado sin comprobar
	// nil, y un default nil revienta el handler con SIGSEGV en vez de fallar
	// con un error legible.
	return &models.RegisterResult{UsuarioID: 1, Email: req.Email, CodigoRef: "DEF001"}, nil
}

func (f *fakeRepo) GetUsuarioByCodigoRef(codigo string) (*models.Referido, error) {
	if f.GetUsuarioByCodigoRefFn != nil {
		return f.GetUsuarioByCodigoRefFn(codigo)
	}
	return nil, nil
}

func (f *fakeRepo) GetReferidoIDByUsuario(usuarioID int) (*int, error) {
	if f.GetReferidoIDByUsuarioFn != nil {
		return f.GetReferidoIDByUsuarioFn(usuarioID)
	}
	return nil, nil
}

func (f *fakeRepo) GetAllProductos() ([]models.Producto, error) {
	if f.GetAllProductosFn != nil {
		return f.GetAllProductosFn()
	}
	return []models.Producto{}, nil
}

func (f *fakeRepo) GetProductoByID(id int) (*models.Producto, error) {
	if f.GetProductoByIDFn != nil {
		return f.GetProductoByIDFn(id)
	}
	return nil, nil
}

func (f *fakeRepo) GetProductosAdmin(filtro interfaces.ProductoFiltro) ([]models.Producto, int, error) {
	if f.GetProductosAdminFn != nil {
		return f.GetProductosAdminFn(filtro)
	}
	return []models.Producto{}, 0, nil
}

func (f *fakeRepo) CreateProducto(d dto.ProductoCreate) (*models.Producto, error) {
	if f.CreateProductoFn != nil {
		return f.CreateProductoFn(d)
	}
	return nil, nil
}

func (f *fakeRepo) UpdateProducto(id int, d dto.ProductoUpdate) (*models.Producto, error) {
	if f.UpdateProductoFn != nil {
		return f.UpdateProductoFn(id, d)
	}
	return nil, nil
}

func (f *fakeRepo) SoftDeleteProducto(id int) error {
	if f.SoftDeleteProductoFn != nil {
		return f.SoftDeleteProductoFn(id)
	}
	return nil
}

func (f *fakeRepo) GetRedGlobal() ([]domain.NodoPlano, error) {
	if f.GetRedGlobalFn != nil {
		return f.GetRedGlobalFn()
	}
	return []domain.NodoPlano{}, nil
}

func (f *fakeRepo) GetRedPropia(usuarioID int) ([]domain.NodoPlano, error) {
	if f.GetRedPropiaFn != nil {
		return f.GetRedPropiaFn(usuarioID)
	}
	return []domain.NodoPlano{}, nil
}

func (f *fakeRepo) CreatePedidoConUsuario(
	req dto.CheckoutRequest, usuarioID int,
) (*models.PedidoDetalle, error) {
	if f.CreatePedidoConUsuarioFn != nil {
		return f.CreatePedidoConUsuarioFn(req, usuarioID)
	}
	return nil, errors.New("no configurado")
}

func (f *fakeRepo) GetMisPedidos(usuarioID, page, limit int) ([]models.PedidoResumen, int, error) {
	if f.GetMisPedidosFn != nil {
		return f.GetMisPedidosFn(usuarioID, page, limit)
	}
	return []models.PedidoResumen{}, 0, nil
}

func (f *fakeRepo) GetPedidosAdmin(filtro interfaces.PedidoFiltro) ([]models.PedidoResumen, int, error) {
	if f.GetPedidosAdminFn != nil {
		return f.GetPedidosAdminFn(filtro)
	}
	return []models.PedidoResumen{}, 0, nil
}

func (f *fakeRepo) GetPedidoDetalle(id int) (*models.PedidoDetalle, error) {
	if f.GetPedidoDetalleFn != nil {
		return f.GetPedidoDetalleFn(id)
	}
	return nil, nil
}

func (f *fakeRepo) CambiarEstadoPedido(id int, nuevoEstado domain.EstadoPedido) error {
	if f.CambiarEstadoPedidoFn != nil {
		return f.CambiarEstadoPedidoFn(id, nuevoEstado)
	}
	return nil
}

func (f *fakeRepo) GetComisiones(filtro interfaces.ComisionFiltro) ([]models.Comision, int, error) {
	if f.GetComisionesFn != nil {
		return f.GetComisionesFn(filtro)
	}
	return []models.Comision{}, 0, nil
}

func (f *fakeRepo) GetTotalesComisiones(referidoID *int) ([]models.ComisionTotal, error) {
	if f.GetTotalesComisionesFn != nil {
		return f.GetTotalesComisionesFn(referidoID)
	}
	return []models.ComisionTotal{}, nil
}

func (f *fakeRepo) ActualizarVentasUpline(emailUsuario string, total float64) error {
	if f.ActualizarVentasUplineFn != nil {
		return f.ActualizarVentasUplineFn(emailUsuario, total)
	}
	return nil
}

func (f *fakeRepo) Liquidar(referidoID int) (*models.Liquidacion, error) {
	if f.LiquidarFn != nil {
		return f.LiquidarFn(referidoID)
	}
	return nil, nil
}

func (f *fakeRepo) GetLiquidaciones(page, limit int) ([]models.Liquidacion, int, error) {
	if f.GetLiquidacionesFn != nil {
		return f.GetLiquidacionesFn(page, limit)
	}
	return []models.Liquidacion{}, 0, nil
}

func (f *fakeRepo) GetLiquidacionesDe(referidoID, page, limit int) ([]models.Liquidacion, int, error) {
	if f.GetLiquidacionesDeFn != nil {
		return f.GetLiquidacionesDeFn(referidoID, page, limit)
	}
	return []models.Liquidacion{}, 0, nil
}

func (f *fakeRepo) GetDeuda() ([]models.DeudaDistribuidor, error) {
	if f.GetDeudaFn != nil {
		return f.GetDeudaFn()
	}
	return []models.DeudaDistribuidor{}, nil
}

func (f *fakeRepo) GetReporteResumen(filtro interfaces.ReporteFiltro) (*models.ReporteResumen, error) {
	if f.GetReporteResumenFn != nil {
		return f.GetReporteResumenFn(filtro)
	}
	return &models.ReporteResumen{}, nil
}

func (f *fakeRepo) GetTopProductos(limit int) ([]models.TopProducto, error) {
	if f.GetTopProductosFn != nil {
		return f.GetTopProductosFn(limit)
	}
	return []models.TopProducto{}, nil
}

func (f *fakeRepo) GetTopReferidos(limit int) ([]models.TopReferido, error) {
	if f.GetTopReferidosFn != nil {
		return f.GetTopReferidosFn(limit)
	}
	return []models.TopReferido{}, nil
}

func (f *fakeRepo) GetComisionesPendientes(limit int) ([]models.Comision, error) {
	if f.GetComisionesPendientesFn != nil {
		return f.GetComisionesPendientesFn(limit)
	}
	return []models.Comision{}, nil
}
