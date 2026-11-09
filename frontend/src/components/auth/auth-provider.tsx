"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { usePathname, useRouter } from "next/navigation";
import { getCurrentUser, logout as logoutRequest } from "@/lib/api/auth";
import { ApiError, setSessionRefreshEnabled } from "@/lib/api/client";
import type { CurrentUser } from "@/lib/api/types";

type AuthContextValue = {
  user: CurrentUser | null;
  refreshUser: () => Promise<CurrentUser>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({
  initialUser,
  children,
}: {
  initialUser: CurrentUser;
  children: ReactNode;
}) {
  const [user, setUser] = useState<CurrentUser | null>(initialUser);
  const router = useRouter();
  const pathname = usePathname();
  const handlingUnauthorized = useRef(false);

  useEffect(() => {
    setSessionRefreshEnabled(true);
  }, []);

  const logout = useCallback(async () => {
    setSessionRefreshEnabled(false);
    try {
      await logoutRequest();
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 401)) {
        setSessionRefreshEnabled(true);
        throw error;
      }
    }
    setUser(null);
    router.replace("/login");
  }, [router]);

  const refreshUser = useCallback(async () => {
    const current = await getCurrentUser();
    setUser(current);
    return current;
  }, []);

  useEffect(() => {
    const onUnauthorized = () => {
      if (handlingUnauthorized.current) return;
      handlingUnauthorized.current = true;
      setSessionRefreshEnabled(false);
      setUser(null);
      if (pathname !== "/login") {
        router.replace(`/login?redirect=${encodeURIComponent(pathname)}`);
      }
      window.setTimeout(() => {
        handlingUnauthorized.current = false;
      }, 1000);
    };
    window.addEventListener("mars:unauthorized", onUnauthorized);
    return () => window.removeEventListener("mars:unauthorized", onUnauthorized);
  }, [pathname, router]);

  const value = useMemo(() => ({ user, refreshUser, logout }), [user, refreshUser, logout]);
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth must be used inside AuthProvider");
  return value;
}
