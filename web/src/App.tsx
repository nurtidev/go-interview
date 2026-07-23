import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { ThemeProvider } from 'next-themes'
import { Toaster } from '@/components/ui/sonner'
import { Layout } from '@/components/Layout'
import { RequireAuth } from '@/components/RequireAuth'
import { AuthProvider, useAuth } from '@/lib/auth'
import LandingPage from '@/pages/LandingPage'
import LoginPage from '@/pages/LoginPage'
import RegisterPage from '@/pages/RegisterPage'
import DashboardPage from '@/pages/DashboardPage'
import SectionPage from '@/pages/SectionPage'
import QuestionPage from '@/pages/QuestionPage'
import ReviewPage from '@/pages/ReviewPage'
import StatsPage from '@/pages/StatsPage'
import LearnPage from '@/pages/LearnPage'
import LessonPage from '@/pages/LessonPage'
import CodeListPage from '@/pages/CodeListPage'
import CodeTaskPage from '@/pages/CodeTaskPage'

function App() {
  return (
    <ThemeProvider attribute="class" defaultTheme="system" enableSystem disableTransitionOnChange>
      <BrowserRouter>
        <AuthProvider>
          <Toaster richColors position="top-center" />
          <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterRoute />} />

          {/* Публичный корень: гость → лендинг (вне Layout); авторизован → Dashboard в Layout. */}
          <Route path="/" element={<HomeRoute />} />

          <Route element={<RequireAuth />}>
            <Route element={<Layout />}>
              <Route path="/learn" element={<LearnPage />} />
              <Route path="/learn/:slug" element={<LessonPage />} />
              <Route path="/code" element={<CodeListPage />} />
              <Route path="/code/:slug" element={<CodeTaskPage />} />
              <Route path="/section/:id" element={<SectionPage />} />
              <Route path="/q/:slug" element={<QuestionPage />} />
              <Route path="/review" element={<ReviewPage />} />
              <Route path="/stats" element={<StatsPage />} />
            </Route>
          </Route>

            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </AuthProvider>
      </BrowserRouter>
    </ThemeProvider>
  )
}

// /register: когда самостоятельная регистрация закрыта на этом инстансе,
// маршрут ведёт на /login с ненавязчивой подписью вместо формы регистрации.
function RegisterRoute() {
  const { registrationEnabled } = useAuth()
  if (!registrationEnabled) {
    return <Navigate to="/login" replace state={{ notice: 'Регистрация закрыта' }} />
  }
  return <RegisterPage />
}

// Корневой маршрут «/»: без токена — публичный лендинг; с токеном — существующая
// связка Layout + Dashboard (как было). RequireAuth здесь не нужен: гостю показываем
// лендинг, а не редирект на /login.
function HomeRoute() {
  const { isAuthenticated } = useAuth()
  if (!isAuthenticated) return <LandingPage />
  return (
    <Layout>
      <DashboardPage />
    </Layout>
  )
}

export default App
