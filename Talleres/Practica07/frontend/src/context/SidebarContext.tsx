import { useState, type ReactNode } from 'react';
import { SidebarContext } from './sidebar';

/**
 * Proveedor del sidebar colapsable.
 * El hook `useSidebar` vive en `sidebar.ts`.
 */
export const SidebarProvider = ({ children }: { children: ReactNode }) => {
  const [isCollapsed, setIsCollapsed] = useState(true);

  const toggleCollapsed = () => {
    setIsCollapsed((prev) => !prev);
  };

  return (
    <SidebarContext.Provider value={{ isCollapsed, toggleCollapsed }}>
      {children}
    </SidebarContext.Provider>
  );
};