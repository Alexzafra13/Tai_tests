// Accounts (internal/users).

export type Role = "admin" | "user";

export type User = {
  id: number;
  username: string;
  display_name: string;
  role: Role;
  active: boolean;
  created_at: string;
};

export const roleLabel: Record<Role, string> = {
  admin: "Administrador",
  user: "Usuario",
};
