import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import { api, ApiError } from "./api";
import type { User } from "./types";

type AuthState = {
  status: "loading" | "in" | "out";
  user: User | null;
  isAdmin: boolean;
  login: (username: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [status, setStatus] = useState<AuthState["status"]>("loading");

  useEffect(() => {
    api<User>("/auth/me")
      .then((u) => {
        setUser(u);
        setStatus("in");
      })
      .catch((err) => {
        if (!(err instanceof ApiError && err.status === 401)) console.error(err);
        setStatus("out");
      });
  }, []);

  const login = useCallback(async (username: string, password: string) => {
    const u = await api<User>("/auth/login", { method: "POST", body: { username, password } });
    setUser(u);
    setStatus("in");
  }, []);

  const logout = useCallback(async () => {
    await api("/auth/logout", { method: "POST" });
    setUser(null);
    setStatus("out");
  }, []);

  return (
    <AuthContext.Provider value={{ status, user, isAdmin: user?.role === "admin", login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}
