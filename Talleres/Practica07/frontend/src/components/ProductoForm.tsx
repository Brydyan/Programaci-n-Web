import { useState, type FormEvent } from 'react';
import { createProducto, updateProducto } from '../services/adminProductosService';
import { ApiError, type Producto, type ProductoCreate } from '../types/api';

interface Props {
  initial?: Producto;
  onCancel: () => void;
  onSaved: (p: Producto) => void;
}

type FieldErrors = Partial<
  Record<'nombre' | 'categoria' | 'precio' | 'stock' | 'img', string>
>;

const URL_RE = /^https?:\/\/\S+\.\S+$/i;

const ProductoForm = ({ initial, onCancel, onSaved }: Props) => {
  const isEdit = Boolean(initial);
  const [nombre, setNombre] = useState<string>(initial?.nombre ?? '');
  const [categoria, setCategoria] = useState<string>(initial?.categoria ?? '');
  const [precio, setPrecio] = useState<string>(
    initial ? String(initial.precio) : '',
  );
  const [stock, setStock] = useState<string>(
    initial ? String(initial.stock) : '',
  );
  const [img, setImg] = useState<string>(initial?.img ?? '');
  const [activo, setActivo] = useState<boolean>(initial?.activo ?? true);

  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});

  const validate = (): FieldErrors => {
    const next: FieldErrors = {};
    if (nombre.trim().length < 2) {
      next.nombre = 'El nombre debe tener al menos 2 caracteres.';
    }
    if (categoria.trim().length < 2) {
      next.categoria = 'La categoría debe tener al menos 2 caracteres.';
    }
    const precioNum = Number(precio);
    if (precio.trim() === '' || Number.isNaN(precioNum)) {
      next.precio = 'Ingresa un precio válido.';
    } else if (precioNum <= 0) {
      next.precio = 'El precio debe ser mayor a 0.';
    }
    const stockNum = Number(stock);
    if (stock.trim() === '' || !Number.isInteger(stockNum) || stockNum < 0) {
      next.stock = 'Ingresa un stock válido (entero ≥ 0).';
    }
    if (img.trim().length > 0 && !URL_RE.test(img.trim())) {
      next.img = 'La URL de la imagen no es válida.';
    }
    return next;
  };

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);
    const next = validate();
    setFieldErrors(next);
    if (Object.keys(next).length > 0) return;

    const basePayload: ProductoCreate = {
      nombre: nombre.trim(),
      categoria: categoria.trim(),
      precio: Number(precio),
      stock: Number(stock),
      img: img.trim() ? img.trim() : undefined,
    };

    setIsLoading(true);
    try {
      const saved = isEdit
        ? await updateProducto(initial!.id, { ...basePayload, activo })
        : await createProducto(basePayload);
      onSaved(saved);
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError(err instanceof Error ? err.message : 'No se pudo guardar el producto.');
      }
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <form
      onSubmit={handleSubmit}
      noValidate
      className="bg-white rounded-lg shadow-md border border-slate-200 w-full max-w-md p-6 space-y-4"
    >
      <h3 className="text-lg font-bold text-slate-900">
        {isEdit ? 'Editar producto' : 'Nuevo producto'}
      </h3>

      {error && (
        <div className="bg-red-50 text-red-600 p-3 rounded-lg text-sm border border-red-200">
          {error}
        </div>
      )}

      <div>
        <label htmlFor="pf-nombre" className="block text-sm font-medium text-slate-700 mb-1">
          Nombre
        </label>
        <input
          id="pf-nombre"
          type="text"
          value={nombre}
          onChange={(e) => setNombre(e.target.value)}
          className="w-full min-h-[44px] px-3 py-2 rounded-lg border border-slate-300 focus:ring-2 focus:ring-marca-600 focus:border-marca-600 outline-none transition"
          aria-invalid={Boolean(fieldErrors.nombre)}
          required
        />
        {fieldErrors.nombre && <p className="mt-1 text-xs text-red-600">{fieldErrors.nombre}</p>}
      </div>

      <div>
        <label htmlFor="pf-categoria" className="block text-sm font-medium text-slate-700 mb-1">
          Categoría
        </label>
        <input
          id="pf-categoria"
          type="text"
          value={categoria}
          onChange={(e) => setCategoria(e.target.value)}
          className="w-full min-h-[44px] px-3 py-2 rounded-lg border border-slate-300 focus:ring-2 focus:ring-marca-600 focus:border-marca-600 outline-none transition"
          aria-invalid={Boolean(fieldErrors.categoria)}
          required
        />
        {fieldErrors.categoria && (
          <p className="mt-1 text-xs text-red-600">{fieldErrors.categoria}</p>
        )}
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div>
          <label htmlFor="pf-precio" className="block text-sm font-medium text-slate-700 mb-1">
            Precio
          </label>
          <input
            id="pf-precio"
            type="number"
            inputMode="decimal"
            step="0.01"
            min="0.01"
            value={precio}
            onChange={(e) => setPrecio(e.target.value)}
            className="w-full min-h-[44px] px-3 py-2 rounded-lg border border-slate-300 focus:ring-2 focus:ring-marca-600 focus:border-marca-600 outline-none transition"
            aria-invalid={Boolean(fieldErrors.precio)}
            required
          />
          {fieldErrors.precio && <p className="mt-1 text-xs text-red-600">{fieldErrors.precio}</p>}
        </div>
        <div>
          <label htmlFor="pf-stock" className="block text-sm font-medium text-slate-700 mb-1">
            Stock
          </label>
          <input
            id="pf-stock"
            type="number"
            inputMode="numeric"
            step="1"
            min="0"
            value={stock}
            onChange={(e) => setStock(e.target.value)}
            className="w-full min-h-[44px] px-3 py-2 rounded-lg border border-slate-300 focus:ring-2 focus:ring-marca-600 focus:border-marca-600 outline-none transition"
            aria-invalid={Boolean(fieldErrors.stock)}
            required
          />
          {fieldErrors.stock && <p className="mt-1 text-xs text-red-600">{fieldErrors.stock}</p>}
        </div>
      </div>

      <div>
        <label htmlFor="pf-img" className="block text-sm font-medium text-slate-700 mb-1">
          URL de imagen <span className="text-slate-400">(opcional)</span>
        </label>
        <input
          id="pf-img"
          type="url"
          value={img}
          onChange={(e) => setImg(e.target.value)}
          className="w-full min-h-[44px] px-3 py-2 rounded-lg border border-slate-300 focus:ring-2 focus:ring-marca-600 focus:border-marca-600 outline-none transition"
          placeholder="https://..."
          aria-invalid={Boolean(fieldErrors.img)}
        />
        {fieldErrors.img && <p className="mt-1 text-xs text-red-600">{fieldErrors.img}</p>}
      </div>

      {isEdit && (
        <label className="flex items-center gap-2 text-sm text-slate-700 select-none">
          <input
            type="checkbox"
            checked={activo}
            onChange={(e) => setActivo(e.target.checked)}
            className="w-4 h-4 accent-marca-600"
          />
          Producto activo
        </label>
      )}

      <div className="flex flex-col-reverse sm:flex-row sm:justify-end gap-2 pt-2">
        <button
          type="button"
          onClick={onCancel}
          disabled={isLoading}
          className="min-h-[44px] px-4 py-2 rounded-lg border border-slate-300 text-slate-700 hover:bg-slate-50 transition disabled:opacity-50"
        >
          Cancelar
        </button>
        <button
          type="submit"
          disabled={isLoading}
          className="min-h-[44px] px-4 py-2 rounded-lg bg-marca-600 text-white font-semibold hover:bg-marca-700 transition disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isLoading ? 'Guardando...' : isEdit ? 'Guardar cambios' : 'Crear producto'}
        </button>
      </div>
    </form>
  );
};

export default ProductoForm;
