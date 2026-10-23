import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vitest/config'
import { VitePWA } from 'vite-plugin-pwa'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    VitePWA({
      // 'prompt' y no 'autoUpdate': si el service worker se autoactualiza
      // mientras el usuario está a mitad de un formulario de checkout, la
      // página puede recargarse y perder lo escrito. El plugin expone
      // `useRegisterSW` (ver components/UpdatePrompt.tsx) para que el
      // usuario aplique la actualización cuando quiera.
      registerType: 'prompt',

      // El service worker estorba al HMR de Vite: sirve versiones viejas
      // del módulo en caliente. En desarrollo va apagado.
      devOptions: { enabled: false },

      includeAssets: ['favicon.svg'],

      manifest: {
        name: 'MultiCatálogo — Tienda Cosmética y MLM',
        short_name: 'MultiCatálogo',
        description:
          'Tienda de cosmética con red de ventas multinivel. Consulta el catálogo y gestiona tus pedidos.',
        lang: 'es-EC',
        dir: 'ltr',
        start_url: '/',
        scope: '/',
        display: 'standalone',
        theme_color: '#1e293b',      // slate-800, el tono de la navbar
        background_color: '#ffffff',

        // Un único icono SVG, el favicon que ya existía en el repo.
        //
        // No se declaran 192/512/maskable en PNG: renombrar un .svg a .png
        // produce un archivo PNG con datos SVG, que Chrome rechaza al validar
        // el manifest. Un SVG con `sizes: 'any'` es válido y es lo que el
        // proyecto ya tenía.
        //
        // `purpose` NO incluye 'maskable': un icono con contenido hasta el
        // borde se ve recortado por la máscara circular de Android. Se
        // declara 'any' a propósito, que es honesto con el arte real.
        icons: [
          {
            src: '/favicon.svg',
            sizes: 'any',
            type: 'image/svg+xml',
            purpose: 'any',
          },
        ],

        // Sin `shortcuts` ni `screenshots`: ambos requieren imágenes que no
        // existen. Declarar rutas rotas hace que la instalación falle en
        // algunos navegadores y baja la nota de Lighthouse.
      },

      workbox: {
        // Precache de lo que produce el build. Los nombres llevan hash
        // (index-D8P9Ohs6.js), y este glob es lo que los descubre. Un
        // service worker escrito a mano no puede adivinarlos, que es
        // exactamente el defecto que inutilizaba el plan anterior.
        globPatterns: ['**/*.{js,css,html,svg,png,ico,woff2}'],

        // SPA: cualquier ruta que no sea un archivo real sirve index.html.
        navigateFallback: 'index.html',
        navigateFallbackDenylist: [/^\/api\//],

        // Las mutaciones NUNCA se cachean ni se reintentan solas.
        // Un POST reenviado automáticamente puede duplicar un pedido.
        globIgnores: ['**/sw.js', '**/workbox-*.js'],

        cleanupOutdatedCaches: true,
        clientsClaim: false,   // coherente con registerType: 'prompt'

        // No se define runtimeCaching todavía: cachear /api/* exige decidir
        // el alcance (sólo catálogo público; nunca datos de sesión). Ver la
        // Fase B de .context/porImplementar/PWA_PLAN.md.
        maximumFileSizeToCacheInBytes: 4 * 1024 * 1024,

        // ── Fase B: offline de LECTURA ────────────────────────────────
        // Sólo el catálogo público. Es la única respuesta de la API que:
        //   a) no depende del token, y
        //   b) es la única que tiene sentido leer sin conexión.
        //
        // Todo lo demás —pedidos, comisiones, red, reportes, liquidaciones—
        // queda en NetworkOnly a propósito. Cachearlo serviría los datos del
        // usuario anterior a quienentrara después en el mismo dispositivo:
        // es una fuga entre sesiones, no una optimización. En el service
        // worker sólo hay reglas para lo que se declara; lo no declarado pasa
        // de largo y va directo a la red.
        runtimeCaching: [
          {
            // `productos` y no `/api/` completo. El patrón es intencionadamente
            // estrecho: prefiero que un endpoint nuevo NO se cachee hasta que
            // alguien decida que debe, que cachear algo por descuido.
            urlPattern: ({ url }) =>
              url.pathname === '/api/productos' ||
              url.pathname === '/api/productos/',
            handler: 'StaleWhileRevalidate',
            options: {
              cacheName: 'catalogo-publico',
              // Tope de entradas para que los query-string de paginación y
              // búsqueda no crezcan el caché sin límite.
              expiration: { maxEntries: 30, maxAgeSeconds: 60 * 60 },
              cacheableResponse: { statuses: [0, 200] },
            },
          },
        ],
      },
    }),
  ],
  server: {
    host: true
  },

  build: {
    rollupOptions: {
      external: ['workbox-window'],
    },
  },

  test: {
    // jsdom y no `node`: los contextos que queremos probar (Cart, Auth)
    // dependen de localStorage, que sólo existe en el DOM.
    environment: 'jsdom',
    globals: false,
    setupFiles: ['./src/test/setup.ts'],
    // Los tests viven junto al código (colocados), no en una carpeta
    // espejo: se lee el módulo y su test de golpe, y borrar un componente
    // borra sus tests sin que queden huérfanos.
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html'],
      // Se miden los archivos con lógica, no los que sólo componen JSX.
      // Un `Navbar.tsx` con 0% no es un hallazgo: no tiene lógica que probar.
      include: [
        'src/context/**/*.{ts,tsx}',
        'src/hooks/**/*.{ts,tsx}',
        'src/services/**/*.{ts,tsx}',
        'src/utils/**/*.{ts,tsx}',
        'src/components/reportes/format.ts',
      ],
    },
  },
})