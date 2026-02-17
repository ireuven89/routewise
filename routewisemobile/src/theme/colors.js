// RouteWiseMobile/src/theme/colors.js
// RouteWise Color Theme - Navy & Orange

export const colors = {
  // Primary
  primary: '#1e3a5f',
  primaryDark: '#152d47',
  primaryLight: '#2d4a6f',
  
  // Accent
  accent: '#ff6b35',
  accentHover: '#ff8555',
  
  // Backgrounds
  background: '#f5f7fa',
  backgroundWhite: '#ffffff',
  inputBg: '#e8eef5',
  
  // Text
  text: '#1a1a1a',
  textSecondary: '#6b7280',
  textLight: '#9ca3af',
  textWhite: '#ffffff',
  
  // Status
  success: '#10b981',
  warning: '#f59e0b',
  error: '#ef4444',
  info: '#3b82f6',
  
  // Status specific
  inProgress: '#ff6b35',
  completed: '#10b981',
  scheduled: '#3b82f6',
  onHold: '#6b7280',
  
  // UI Elements
  border: '#e5e7eb',
  disabled: '#d1d5db',
  shadow: 'rgba(0, 0, 0, 0.1)',
};

export const theme = {
  colors,
  
  spacing: {
    xs: 4,
    sm: 8,
    md: 16,
    lg: 24,
    xl: 32,
  },
  
  borderRadius: {
    sm: 6,
    md: 8,
    lg: 12,
    full: 999,
  },
  
  fontSize: {
    xs: 12,
    sm: 14,
    md: 16,
    lg: 18,
    xl: 24,
    xxl: 32,
  },
  
  shadow: {
    sm: {
      shadowColor: '#000',
      shadowOffset: { width: 0, height: 1 },
      shadowOpacity: 0.1,
      shadowRadius: 3,
      elevation: 2,
    },
    md: {
      shadowColor: '#000',
      shadowOffset: { width: 0, height: 2 },
      shadowOpacity: 0.1,
      shadowRadius: 4,
      elevation: 4,
    },
  },
};

export default theme;
