import { useInstall } from "../../pwa";

// InstallSection explains how to install the app on this device, which
// depends on the browser and on the app being served over HTTPS.
export function InstallSection() {
  const state = useInstall();
  return (
    <fieldset>
      <legend>Instalar la app</legend>
      {state.kind === "installed" && <p className="hint">Ya la estás usando como app instalada.</p>}
      {state.kind === "prompt" && (
        <>
          <p className="hint">Tendrás su icono en la pantalla de inicio y se abrirá a pantalla completa.</p>
          <button type="button" className="primary" onClick={state.install}>
            Instalar
          </button>
        </>
      )}
      {state.kind === "ios" && (
        <p className="hint">
          En Safari, pulsa <strong>Compartir</strong> y luego <strong>Añadir a pantalla de inicio</strong>.
        </p>
      )}
      {state.kind === "needs-https" && (
        <p className="hint">
          Para instalarla, el navegador exige abrir la app por HTTPS (por ejemplo con Tailscale o un proxy inverso).
          Mientras tanto puedes usarla desde el navegador o añadir un acceso directo desde su menú.
        </p>
      )}
      {state.kind === "browser-menu" && (
        <p className="hint">
          Usa el menú del navegador: <strong>Instalar app</strong> o <strong>Añadir a pantalla de inicio</strong>.
        </p>
      )}
    </fieldset>
  );
}
