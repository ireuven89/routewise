// RouteWise Theme - Add to src/theme.js

export const colors = {
  primary: '#1e3a5f',
  accent: '#ff6b35',
  lightBg: '#f5f7fa',
  inputBg: '#e8eef5',
  white: '#ffffff',
  text: '#1a1a1a',
  textGray: '#6b7280',
  success: '#10b981',
  info: '#3b82f6',
  error: '#ef4444',
  warning: '#f59e0b',
};

export const theme = {
  colors,
  
  // Status colors
  status: {
    inProgress: colors.accent,
    completed: colors.success,
    scheduled: colors.info,
    onHold: colors.textGray,
  },
  
  // Shadows
  shadow: {
    sm: '0 1px 3px rgba(0,0,0,0.1)',
    md: '0 4px 6px rgba(0,0,0,0.1)',
  },
  
  // Border radius
  radius: {
    sm: '6px',
    md: '8px',
    lg: '12px',
  },
};

export default theme;
