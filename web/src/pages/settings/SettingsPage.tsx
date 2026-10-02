import { Link } from "react-router";
import { useAuth } from "../../auth";
import { AccountSection } from "./AccountSection";
import { AppearanceSection } from "./AppearanceSection";
import { InstallSection } from "./InstallSection";
import { ScoringSection } from "./ScoringSection";

export function SettingsPage() {
  const { isAdmin, logout } = useAuth();
  return (
    <div className="form">
      <h2>Ajustes</h2>
      <AccountSection />
      <AppearanceSection />
      <InstallSection />
      <ScoringSection editable={isAdmin} />
      {isAdmin && (
        <fieldset>
          <legend>Administración</legend>
          <Link to="/users">Usuarios y permisos</Link>
          <Link to="/sources">Fuentes</Link>
        </fieldset>
      )}
      <button type="button" className="danger" onClick={() => logout()}>
        Cerrar sesión
      </button>
    </div>
  );
}
