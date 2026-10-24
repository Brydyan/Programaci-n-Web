import { createContext, useContext } from 'react';

/**
 * Tipos, objeto de contexto y hook del sidebar.
 * Separado de `SidebarContext.tsx` (sólo Provider) para el Fast Refresh de Vite.
 */

export interface SidebarContextType {
  isCollapsed: boolean;
  toggleCollapsed: () => void;
}

export const SidebarContext = createContext<SidebarContextType | undefined>(undefined);

/** Acceso al estado del sidebar. Lanza si se usa fuera del `SidebarProvider`. */
export const useSidebar = (): SidebarContextType => {
  const context = useContext(SidebarContext);
  if (!context) {
    throw new Error('useSidebar debe ser usado dentro de SidebarProvider');
  }
  return context;
};