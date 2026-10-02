import { useState } from "react";
import { Field } from "../../components/Form";
import { applyTheme, savedTheme, themeLabel, type Theme } from "../../theme";

// AppearanceSection picks the colour theme on this device.
export function AppearanceSection() {
  const [theme, setTheme] = useState(savedTheme);
  function choose(t: Theme) {
    setTheme(t);
    applyTheme(t);
  }
  return (
    <fieldset>
      <legend>Apariencia</legend>
      <Field label="Tema">
        <div className="segmented">
          {(Object.keys(themeLabel) as Theme[]).map((t) => (
            <button key={t} type="button" className={t === theme ? "active" : ""} onClick={() => choose(t)}>
              {themeLabel[t]}
            </button>
          ))}
        </div>
      </Field>
    </fieldset>
  );
}
