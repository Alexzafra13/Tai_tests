import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, Navigate, Outlet, Route, Routes } from "react-router";
import { AuthProvider, useAuth } from "./auth";
import { Layout } from "./Layout";
import { LoginPage } from "./pages/LoginPage";
import { SetupPage } from "./pages/SetupPage";
import { HomePage } from "./pages/HomePage";
import { QuestionsPage } from "./pages/QuestionsPage";
import { QuestionEditPage } from "./pages/QuestionEditPage";
import { SourcesPage } from "./pages/SourcesPage";
import { SourceEditPage } from "./pages/SourceEditPage";
import { SyllabusPage } from "./pages/SyllabusPage";
import { NewTestPage } from "./pages/NewTestPage";
import { TestPage } from "./pages/test/TestPage";
import { TestsPage } from "./pages/TestsPage";
import { SettingsPage } from "./pages/settings/SettingsPage";
import { UsersPage } from "./pages/users/UsersPage";
import { ReviewPage } from "./pages/review/ReviewPage";
import { BatchReviewPage } from "./pages/review/BatchReviewPage";
import "./styles.css";

// AdminOnly hides administration screens from other users. The server
// enforces the same rules; this only avoids showing pages that would fail.
function AdminOnly() {
  const { isAdmin } = useAuth();
  return isAdmin ? <Outlet /> : <Navigate to="/" replace />;
}

function App() {
  const { status } = useAuth();

  if (status === "loading") {
    return <div className="center muted">Cargando…</div>;
  }
  if (status === "setup") {
    return <SetupPage />;
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
        <Route path="tests" element={<TestsPage />} />
        <Route path="tests/new" element={<NewTestPage />} />
        <Route path="tests/:id" element={<TestPage />} />
        <Route path="syllabus" element={<SyllabusPage />} />
        <Route path="settings" element={<SettingsPage />} />

        <Route element={<AdminOnly />}>
          <Route path="questions" element={<QuestionsPage />} />
          <Route path="questions/new" element={<QuestionEditPage />} />
          <Route path="questions/:id" element={<QuestionEditPage />} />
          <Route path="sources" element={<SourcesPage />} />
          <Route path="sources/new" element={<SourceEditPage />} />
          <Route path="sources/:id" element={<SourceEditPage />} />
          <Route path="review" element={<ReviewPage />} />
          <Route path="review/batch" element={<BatchReviewPage />} />
          <Route path="users" element={<UsersPage />} />
        </Route>

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
