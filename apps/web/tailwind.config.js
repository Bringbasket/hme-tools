/** GoKeep 主题 token 与组件基础色。 */
import type { Config } from 'tailwindcss'

export default {
  darkMode: 'class',
  content: ['./index.html', './src/**/*.{vue,ts}'],
  theme: {
    extend: {
      colors: {
        // 品牌主色（按钮/主操作）
        brand: {
          DEFAULT: '#008f83',
          hover: '#08776f',
          border: '#0f8f86',
          50: '#f0fdfa',
          100: '#ccfbf1',
          200: '#99f6e4',
          300: '#5eead4',
          400: '#2dd4bf',
          500: '#14b8a6',
          600: '#0d9488',
          700: '#0f766e',
          800: '#115e59',
        },
        // 顶部渐变条：teal-500 → cyan-500 → emerald-500
        'gradient-bar': {
          from: '#14b8a6',
          via: '#06b6d4',
          to: '#10b981',
        },
        // 语义色（06-UI设计规范：成功/警告/错误/信息）
        surface: {
          page: '#f8fafc',
          panel: '#ffffff',
        },
      },
      fontFamily: {
        sans: [
          'system-ui',
          '-apple-system',
          'BlinkMacSystemFont',
          '"Segoe UI"',
          'Roboto',
          '"Helvetica Neue"',
          'Arial',
          '"PingFang SC"',
          '"Hiragino Sans GB"',
          '"Microsoft YaHei"',
          'sans-serif',
        ],
      },
      boxShadow: {
        floating: '0 24px 70px rgba(15, 23, 42, 0.14)',
        'soft-sm': '0 1px 2px rgba(15, 23, 42, 0.05)',
        brand: '0 8px 16px rgba(0, 143, 131, 0.14)',
      },
      keyframes: {
        'slide-up': {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
      },
      animation: {
        'slide-up': 'slide-up 0.3s ease-out',
      },
    },
  },
  plugins: [],
} satisfies Config
