const defaultTheme = require("tailwindcss/defaultTheme");
const colors = require("tailwindcss/colors");

module.exports = {
  content: [
    "./internal/admin/templates/**/*.templ",
    "./internal/admin/templates/**/*.go",
    "./internal/admin/httpserver/**/*.go",
    "./web/styles/**/*.{css,scss}",
    "./public/**/*.html"
  ],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        brand: {
          25: "#fff5f5",
          50: "#ffe9e8",
          100: "#ffd0cc",
          200: "#ffb0a6",
          300: "#ff8f80",
          400: "#ff6955",
          500: "#f43f2e",
          600: "#dc2916",
          700: "#b81f10",
          800: "#8f1a11",
          900: "#741a16",
          950: "#420a05"
        },
        surface: {
          DEFAULT: "#ffffff",
          subtle: "#f8fafc",
          muted: "#f1f5f9",
          raised: "#ffffff",
          inverted: "#0f172a",
          overlay: "rgba(15, 23, 42, 0.55)"
        },
        border: {
          subtle: "#e2e8f0",
          bold: "#cbd5f5",
          inverted: "#1e293b"
        },
        success: colors.emerald,
        danger: colors.rose,
        warning: colors.amber,
        info: colors.sky
      },
      fontFamily: {
        sans: ["\"Inter\"", ...defaultTheme.fontFamily.sans],
        display: ["\"Lexend\"", ...defaultTheme.fontFamily.sans]
      },
      boxShadow: {
        focus: "0 0 0 4px rgba(244, 63, 46, 0.15)",
        surface: "0 8px 20px rgba(15, 23, 42, 0.08)",
        modal: "0 24px 60px rgba(15, 23, 42, 0.20)",
        toast: "0 14px 40px rgba(15, 23, 42, 0.12)"
      },
      borderRadius: {
        xl: "1rem",
        "2xl": "1.25rem"
      },
      spacing: {
        13: "3.25rem",
        15: "3.75rem",
        18: "4.5rem"
      },
      keyframes: {
        "dialog-in": {
          "0%": { opacity: "0", transform: "translateY(16px) scale(0.98)" },
          "100%": { opacity: "1", transform: "translateY(0) scale(1)" }
        },
        "dialog-out": {
          "0%": { opacity: "1", transform: "translateY(0) scale(1)" },
          "100%": { opacity: "0", transform: "translateY(8px) scale(0.98)" }
        },
        "toast-in": {
          "0%": { opacity: "0", transform: "translateY(12px)" },
          "100%": { opacity: "1", transform: "translateY(0)" }
        },
        "toast-out": {
          "0%": { opacity: "1", transform: "translateY(0)" },
          "100%": { opacity: "0", transform: "translateY(12px)" }
        }
      },
      animation: {
        "dialog-in": "dialog-in 180ms cubic-bezier(0.16, 1, 0.3, 1) forwards",
        "dialog-out": "dialog-out 140ms ease-in forwards",
        "toast-in": "toast-in 140ms ease-out forwards",
        "toast-out": "toast-out 180ms ease-in forwards"
      }
    }
  },
  plugins: []
};
