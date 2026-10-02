import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import { api, ApiError } from "./api";
import type { User } from "./types";

// status "setup" means a fresh install with no administrator yet: the app
// shows the first-run setup screen instead of the login. "offline" means
// the server could not be reached, which says nothing about the session.
type AuthState = {
  status: "loading" | "setup" | "in" | "out" | "offline";
  retry: () => void;
  user: User | null;
  isAdmin: boolean;
  login: (username: string, password: string) => Promise<void>;
  setup: (input: SetupInput) => Promise<void>;
  logout: () => Promise<void>;
};

export type SetupInput = { username: string; display_name: string; password: string };

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [status, setStatus] = useState<AuthState["status"]>("loading");

  const check = useCallback(() => {
    api<User>("/auth/me")
      .then((u) => {
        setUser(u);
        setStatus("in");
      })
      .catch(async (err) => {
        if (err instanceof ApiError && err.status === 0) {
          setStatus("offline");
          return;
        }
        if (!(err instanceof ApiError && err.status === 401)) console.error(err);
        const setup = await api<{ needed: boolean }>("/setup").catch(() => ({ needed: false }));
        setStatus(setup.needed ? "setup" : "out");
      });
  }, []);

  useEffect(check, [check]);

  // Opened without a connection: try again as soon as it comes back.
  useEffect(() => {
    if (status !== "offline") return;
    window.addEventListener("online", check);
    return () => window.removeEventListener("online", check);
  }, [status, check]);

  const login = useCallback(async (username: string, password: string) => {
    const u = await api<User>("/auth/login", { method: "POST", body: { username, password } });
    setUser(u);
    setStatus("in");
  }, []);

  const setup = useCallback(async (input: SetupInput) => {
    const u = await api<User>("/setup", { method: "POST", body: input });
    setUser(u);
    setStatus("in");
  }, []);

  const logout = useCallback(async () => {
    await api("/auth/logout", { method: "POST" });
    setUser(null);
    setStatus("out");
  }, []);

  return (
    <AuthContext.Provider value={{ status, retry: check, user, isAdmin: user?.role === "admin", login, setup, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}
