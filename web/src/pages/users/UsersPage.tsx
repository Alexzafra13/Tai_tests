import { useState } from "react";
import { useAuth } from "../../auth";
import { useResource } from "../../hooks";
import type { User } from "../../types";
import { ErrorBox, Loading } from "../../components/Form";
import { NewUserForm } from "./NewUserForm";
import { UserCard } from "./UserCard";

export function UsersPage() {
  const { user: me } = useAuth();
  const users = useResource<User[]>("/users");
  const [adding, setAdding] = useState(false);

  return (
    <div className="form">
      <div className="page-head">
        <h2>Usuarios</h2>
        {!adding && (
          <button type="button" className="primary small" onClick={() => setAdding(true)}>
            Nuevo
          </button>
        )}
      </div>
      <p className="hint">
        Los usuarios estudian y marcan dudas. Los administradores además gestionan preguntas, fuentes, revisión, nota y
        cuentas. No hay registro abierto: las cuentas se crean aquí.
      </p>

      {adding && (
        <NewUserForm
          onCreated={() => {
            setAdding(false);
            users.reload();
          }}
        />
      )}

      <ErrorBox message={users.error} />
      {users.loading && !users.data && <Loading />}
      <ul className="list">
        {users.data?.map((u) => (
          <UserCard key={u.id} user={u} isSelf={u.id === me?.id} onChange={users.reload} />
        ))}
      </ul>
    </div>
  );
}
