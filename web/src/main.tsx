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
import { NewTestPage } from "./pages/NewTestPage";
import { TestPage } from "./pages/test/TestPage";
import { TestsPage } from "./pages/TestsPage";
import { SettingsPage } from "./pages/SettingsPage";
import { ReviewPage } from "./pages/review/ReviewPage";
import { BatchReviewPage } from "./pages/review/BatchReviewPage";
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
        <Route path="tests" element={<TestsPage />} />
        <Route path="tests/new" element={<NewTestPage />} />
        <Route path="tests/:id" element={<TestPage />} />
        <Route path="settings" element={<SettingsPage />} />
        <Route path="review" element={<ReviewPage />} />
        <Route path="review/batch" element={<BatchReviewPage />} />
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
