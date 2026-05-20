const DEFAULT_UPSTREAM = "https://api.xiaoheihe.cn";

function normalizePrefix(prefix) {
  const value = (prefix || "").trim();
  if (!value || value === "/") {
    return "";
  }
  return value.startsWith("/") ? value.replace(/\/+$/, "") : `/${value.replace(/\/+$/, "")}`;
}

function forwardedHeaders(request) {
  const headers = new Headers(request.headers);
  for (const name of [
    "host",
    "cf-connecting-ip",
    "cf-ipcountry",
    "cf-ray",
    "cf-visitor",
    "x-forwarded-for",
    "x-forwarded-proto",
    "x-real-ip",
  ]) {
    headers.delete(name);
  }
  return headers;
}

export default {
  async fetch(request, env) {
    if (!["GET", "POST", "HEAD"].includes(request.method)) {
      return new Response("Method not allowed", { status: 405 });
    }

    const prefix = normalizePrefix(env.PROXY_PATH_PREFIX);
    if (!prefix) {
      return new Response("Missing PROXY_PATH_PREFIX", { status: 500 });
    }

    const url = new URL(request.url);
    if (url.pathname !== prefix && !url.pathname.startsWith(`${prefix}/`)) {
      return new Response("Not found", { status: 404 });
    }

    const upstreamBase = new URL(env.UPSTREAM_ORIGIN || DEFAULT_UPSTREAM);
    const upstream = new URL(upstreamBase);
    upstream.pathname = url.pathname.slice(prefix.length) || "/";
    upstream.search = url.search;

    const init = {
      method: request.method,
      headers: forwardedHeaders(request),
      redirect: "manual",
    };
    if (!["GET", "HEAD"].includes(request.method)) {
      init.body = request.body;
    }

    return fetch(upstream.toString(), init);
  },
};
