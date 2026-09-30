import { Link } from "react-router";
import { useAuth } from "../../auth";
import { AccountSection } from "./AccountSection";
import { ScoringSection } from "./ScoringSection";

export function SettingsPage() {
  const { isAdmin } = useAuth();
  return (
    <div className="form">
      <h2>Ajustes</h2>
      <AccountSection />
      <ScoringSection editable={isAdmin} />
      {isAdmin && (
        <fieldset>
          <legend>Administración</legend>
          <Link to="/users">Usuarios y permisos</Link>
          <Link to="/sources">Fuentes</Link>
        </fieldset>
      )}
    </div>
  );
}
