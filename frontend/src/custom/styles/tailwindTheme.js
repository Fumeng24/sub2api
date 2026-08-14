export function applyWegooTailwindTheme(config) {
  const extend = config.theme.extend
  extend.colors.primary = {
    50: '#eff6ff',
    100: '#dbeafe',
    200: '#bfdbfe',
    300: '#93c5fd',
    400: '#2997ff',
    500: '#0077ed',
    600: '#0071e3',
    700: '#0066cc',
    800: '#004a99',
    900: '#003b7a',
    950: '#001f40'
  }
  extend.colors.dark = {
    50: '#f5f5f7',
    100: '#e8e8ed',
    200: '#d2d2d7',
    300: '#a1a1a6',
    400: '#86868b',
    500: '#6e6e73',
    600: '#3a3a3c',
    700: '#2c2c2e',
    800: '#1d1d1f',
    900: '#111113',
    950: '#000000'
  }
  Object.assign(extend.boxShadow, {
    glass: '0 1px 2px rgba(0, 0, 0, 0.04)',
    'glass-sm': '0 1px 2px rgba(0, 0, 0, 0.04)',
    glow: '0 1px 2px rgba(0, 0, 0, 0.04)',
    'glow-lg': '0 10px 24px rgba(0, 0, 0, 0.08)'
  })
  Object.assign(extend.backgroundImage, {
    'gradient-primary': 'linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%)',
    'mesh-gradient':
      'radial-gradient(at 40% 20%, rgba(37, 99, 235, 0.1) 0px, transparent 50%), radial-gradient(at 80% 0%, rgba(14, 165, 233, 0.08) 0px, transparent 50%), radial-gradient(at 0% 50%, rgba(245, 158, 11, 0.07) 0px, transparent 50%)'
  })
  extend.keyframes.glow = {
    '0%': { boxShadow: '0 0 20px rgba(37, 99, 235, 0.22)' },
    '100%': { boxShadow: '0 0 30px rgba(37, 99, 235, 0.34)' }
  }
  return config
}
