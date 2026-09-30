const sections = [
  { title: "Test", desc: "Práctica, examen y simulacros oficiales" },
  { title: "Repaso", desc: "Repetición espaciada y preguntas falladas" },
  { title: "Revisión", desc: "Borradores y preguntas marcadas como dudosas" },
  { title: "Estadísticas", desc: "Aciertos por bloque y tema" },
];

export function HomePage() {
  return (
    <>
      <h2>Inicio</h2>
      <ul className="tiles">
        {sections.map((s) => (
          <li key={s.title} className="card tile disabled">
            <strong>{s.title}</strong>
            <span className="muted">{s.desc}</span>
            <span className="badge">Próximamente</span>
          </li>
        ))}
      </ul>
    </>
  );
}
