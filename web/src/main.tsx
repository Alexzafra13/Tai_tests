import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, Navigate, Route, Routes } from "react-router";
import { AuthProvider, useAuth } from "./auth";
import { Layout } from "./Layout";
import { LoginPage } from "./pages/LoginPage";
import { HomePage } from "./pages/HomePage";
import { QuestionsPage } from "./pages/QuestionsPage";
import { QuestionEditPage } from "./pages/QuestionEditPage";
import { SourcesPage } from "./pages/SourcesPage";
import { SourceEditPage } from "./pages/SourceEditPage";
import { SyllabusPage } from "./pages/SyllabusPage";
import "./styles.css";

function App() {
  const { status } = useAuth();

  if (status === "loading") {
    return <div className="center muted">Cargando…</div>;
  }
  if (status === "out") {
    return (
      <Routes>
        <Route path="*" element={<LoginPage />} />
      </Routes>
    );
  }
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<HomePage />} />
        <Route path="questions" element={<QuestionsPage />} />
        <Route path="questions/new" element={<QuestionEditPage />} />
        <Route path="questions/:id" element={<QuestionEditPage />} />
        <Route path="sources" element={<SourcesPage />} />
        <Route path="sources/new" element={<SourceEditPage />} />
        <Route path="sources/:id" element={<SourceEditPage />} />
        <Route path="syllabus" element={<SyllabusPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <BrowserRouter>
      <AuthProvider>
        <App />
      </AuthProvider>
    </BrowserRouter>
  </StrictMode>,
);
