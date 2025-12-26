import { jsx, jsxs } from "react/jsx-runtime";
function StatsRow() {
  return /* @__PURE__ */ jsx("div", { className: "grid gap-4 md:grid-cols-4" });
}
function Dashboard() {
  return /* @__PURE__ */ jsx("div", { className: "flex flex-1 flex-col gap-4 p-4", children: /* @__PURE__ */ jsxs("div", { className: "mx-auto w-full max-w-7xl space-y-8", children: [
    /* @__PURE__ */ jsxs("header", { children: [
      /* @__PURE__ */ jsx("h1", { className: "text-3xl font-bold tracking-tight", children: "Arbitrage Dashboard" }),
      /* @__PURE__ */ jsx("p", { className: "text-muted-foreground", children: "Polymarket ↔ Kalshi performance overview" })
    ] }),
    /* @__PURE__ */ jsx(StatsRow, {})
  ] }) });
}
export {
  Dashboard as component
};
