import { NextResponse, type NextRequest } from "next/server";

export function proxy(request: NextRequest): NextResponse {
  const headers = new Headers(request.headers);
  headers.set("x-mars-pathname", `${request.nextUrl.pathname}${request.nextUrl.search}`);
  return NextResponse.next({ request: { headers } });
}

export const config = {
  matcher: ["/((?!api|_next/static|_next/image|favicon.ico|.*\\..*).*)"],
};
