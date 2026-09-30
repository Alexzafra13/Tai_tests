import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import { api, ApiError } from "./api";

type AuthState = {
  status: "loading" | "in" | "out";
  login: (password: string) => Promise<void>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthState["status"]>("loading");

  useEffect(() => {
    api("/auth/me")
      .then(() => setStatus("in"))
      .catch((err) => {
        if (!(err instanceof ApiError && err.status === 401)) console.error(err);
        setStatus("out");
      });
  }, []);

  const login = useCallback(async (password: string) => {
    await api("/auth/login", { method: "POST", body: { password } });
    setStatus("in");
  }, []);

  const logout = useCallback(async () => {
    await api("/auth/logout", { method: "POST" });
    setStatus("out");
  }, []);

  return <AuthContext.Provider value={{ status, login, logout }}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}
