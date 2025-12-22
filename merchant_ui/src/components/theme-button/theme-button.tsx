import { Moon, Sun } from "lucide-react";
import { Button } from "@/components/ui/button";
import { getTheme, setTheme } from "@/lib/theme";
import { useEffect, useState } from "react";

export function ThemeToggle() {
  const [theme, setCurrentTheme] = useState(getTheme());

  useEffect(() => {
    setCurrentTheme(getTheme());
  }, []);

  function toggleTheme() {
    const next =
      theme === "tokyo-dark" ? "tokyo-light" : "tokyo-dark";
    setTheme(next);
    setCurrentTheme(next);
  }

  return (
    <Button
      variant="ghost"
      size="icon"
      onClick={toggleTheme}
      aria-label="Toggle theme"
    >
      {theme === "tokyo-dark" ? (
        <Sun className="h-4 w-4" />
      ) : (
        <Moon className="h-4 w-4" />
      )}
    </Button>
  );
}
