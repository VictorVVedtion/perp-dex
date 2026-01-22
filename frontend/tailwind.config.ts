import type { Config } from 'tailwindcss'

const config: Config = {
  content: [
    './src/pages/**/*.{js,ts,jsx,tsx,mdx}',
    './src/components/**/*.{js,ts,jsx,tsx,mdx}',
    './src/app/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#e6fff7',
          100: '#ccffef',
          200: '#99ffdf',
          300: '#66ffcf',
          400: '#33ffbf',
          500: '#00C896',
          600: '#00a078',
          700: '#00785a',
          800: '#00503c',
          900: '#00281e',
        },
        danger: {
          50: '#fff0f0',
          100: '#ffe1e1',
          200: '#ffc2c2',
          300: '#ffa4a4',
          400: '#ff8585',
          500: '#FF4D4D',
          600: '#cc3d3d',
          700: '#992e2e',
          800: '#661f1f',
          900: '#330f0f',
        },
        dark: {
          50: '#f8fafc',
          100: '#f1f5f9',
          200: '#e2e8f0',
          300: '#cbd5e1',
          400: '#94a3b8',
          500: '#64748b',
          600: '#475569',
          700: '#334155',
          800: '#1e293b',
          900: '#0f172a',
          950: '#020617',
        }
      },
      backgroundImage: {
        'primary-gradient': 'linear-gradient(135deg, #00C896 0%, #008f6b 100%)',
        'danger-gradient': 'linear-gradient(135deg, #FF4D4D 0%, #cc3d3d 100%)',
      },
      boxShadow: {
        'glow-primary': '0 0 15px rgba(0, 200, 150, 0.3)',
        'glow-danger': '0 0 15px rgba(255, 77, 77, 0.3)',
        'glow-sm': '0 0 10px rgba(0, 200, 150, 0.2)',
        'glow-md': '0 0 20px rgba(0, 200, 150, 0.3)',
        'glow-lg': '0 0 30px rgba(0, 200, 150, 0.4)',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'Menlo', 'Monaco', 'monospace'],
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-down': 'slideDown 0.3s ease-out',
        'pulse-glow': 'pulseGlow 2s infinite',
        'number-up': 'numberUp 1s ease-out',
        'number-down': 'numberDown 1s ease-out',
        'shimmer': 'shimmer 2s infinite linear',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideUp: {
          '0%': { transform: 'translateY(10px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
        slideDown: {
          '0%': { transform: 'translateY(-10px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
        pulseGlow: {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '0.7' },
        },
        numberUp: {
          '0%': { color: '#00C896' },
          '100%': { color: 'inherit' },
        },
        numberDown: {
          '0%': { color: '#FF4D4D' },
          '100%': { color: 'inherit' },
        },
        shimmer: {
          '100%': { transform: 'translateX(100%)' },
        }
      }
    },
  },
  plugins: [],
}
export default config
