import { createContext, useContext } from 'react';

// { mode: 'customer' | 'technician' | null, setMode(mode) } — provided by AppNavigator.
export const ModeContext = createContext({ mode: null, setMode: () => {} });

export const useMode = () => useContext(ModeContext);
