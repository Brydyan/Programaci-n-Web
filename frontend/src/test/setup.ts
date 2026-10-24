// Configuración global de Vitest.
//
// Se ejecuta una vez por archivo de test, antes de que se importe nada.

import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';

// Sin esto, el DOM de un test se acumula en el siguiente y los `getByText`
// encuentran nodos de tests anteriores. El error clásico de "funciona solo
// cuando lo ejecuto en aislamiento".
afterEach(() => {
  cleanup();
  localStorage.clear();
});
