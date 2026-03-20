import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

const publicPaths = ["/login"];

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Allow public paths
  if (publicPaths.some((p) => pathname.startsWith(p))) {
    return NextResponse.next();
  }

  // Allow static assets, API proxy, and Next.js internals
  const STATIC_EXT = /\.(ico|png|jpg|jpeg|gif|svg|css|js|woff2?|ttf|eot|map|webp|avif)$/;
  if (
    pathname.startsWith("/api") ||
    pathname.startsWith("/_next") ||
    STATIC_EXT.test(pathname)
  ) {
    return NextResponse.next();
  }

  // Check for auth token in cookie (set by frontend on login)
  const token = request.cookies.get("access_token")?.value;
  if (!token) {
    const loginUrl = new URL("/login", request.url);
    loginUrl.searchParams.set("redirect", pathname);
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
