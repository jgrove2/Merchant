export type Theme = "tokyo-dark" | "tokyo-light";

const THEME_KEY = "theme";

export function setTheme(theme: Theme) {
  if (typeof window === "undefined") return;
  document.documentElement.setAttribute("data-theme", theme);
  localStorage.setItem(THEME_KEY, theme);
}

export function getTheme(): Theme {
  if (typeof window === "undefined") return "tokyo-dark";
  return (
    (localStorage.getItem(THEME_KEY) as Theme) || "tokyo-dark"
  );
}

export function initTheme() {
  if (typeof window === "undefined") return;
  setTheme(getTheme());
}
