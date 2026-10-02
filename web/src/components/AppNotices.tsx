import { useAuth } from "../auth";
import { useOnline, useUpdate } from "../pwa";
import "./AppNotices.css";

// AppNotices shows app-wide states: no connection, and a new version ready
// to use. Updating reloads the page; a test in progress resumes where it
// was, because every answer is already saved on the server.
export function AppNotices() {
  const online = useOnline();
  // Opened offline, the whole screen already says so.
  const { status } = useAuth();
  const applyUpdate = useUpdate();
  return (
    <>
      {!online && status !== "offline" && (
        <div className="offline-bar" role="status">
          Sin conexión. Las respuestas no se guardan hasta que vuelva.
        </div>
      )}
      {applyUpdate && (
        <div className="update-bar" role="status">
          <span>Hay una versión nueva de la app.</span>
          <button type="button" className="link" onClick={applyUpdate}>
            Actualizar
          </button>
        </div>
      )}
    </>
  );
}
