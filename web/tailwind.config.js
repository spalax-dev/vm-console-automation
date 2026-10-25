/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,html}",
  ],
  theme: {
    extend: {
      colors: {
        vm: {
          bg: '#F0F0F0',
          surface: '#FFFFFF',
          border: '#B0B0B0',
          borderHover: '#C0392B',
          accent: '#C0392B',
          fg: '#1F1F1F',
          fgSecondary: '#666666',
          success: '#27AE60',
          warning: '#F39C12',
          muted: '#7F8C8D',
        }
      },
      borderRadius: {
        vm: '3px',
        'vm-md': '4px',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
      height: {
        'vm-header': '40px',
        'vm-tabs': '30px',
        'vm-btn': '26px',
        'vm-input': '26px',
      },
      spacing: {
        'vm-gap': '16px',
      },
      maxWidth: {
        'vm': '1100px',
      },
    },
  },
  plugins: [require('@tailwindcss/forms')],
}
