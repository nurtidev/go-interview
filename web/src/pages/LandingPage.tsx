import { useRef, useState, type ReactNode } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { MenuIcon, XIcon } from 'lucide-react'
import { plural } from '@/lib/format'

/* ============================================================================
   Лендинг GoPrep — маршрут «/» для гостей (см. design_handoff_goprep/LANDING.md
   и референс «GoPrep Лендинг.dc.html»). Живёт вне Layout приложения: свой
   glass-навбар. Пиксельно по референсу; отступления от мокапа помечены [ОТКЛ].

   Стекло — строго в 4 местах (навбар-пилюля, hero-карточка, пилюля таймера,
   карточки отзывов) через классы .lp-glass-* в index.css + fallback @supports.
   Светлые значения токенов пиннятся классом .goprep-landing → лендинг остаётся
   светлым даже при `.dark` на <html>.
   ========================================================================== */

const NAV_LINKS = [
  { label: 'Как это работает', id: 'how' },
  { label: 'Программа', id: 'program' },
  { label: 'Отзывы', id: 'testimonials' },
  { label: 'FAQ', id: 'faq' },
] as const

// [ОТКЛ №1] Реальные числа продукта вместо демо-цифр мокапа (156/24/15 мин).
// 84 = сумма вопросов по 7 секциям ниже.
const HERO_METRICS = [
  { value: '84', label: 'вопросов с секций' },
  { value: '22', label: 'задачи лайвкодинга' },
  { value: '10', label: 'глав со схемами' },
] as const

// [ОТКЛ №1] Реальные 7 секций программы (в мокапе было 6 демо-тем).
const PROGRAM_SECTIONS = [
  { title: 'Go изнутри', count: 15 },
  { title: 'Конкурентность', count: 12 },
  { title: 'Алгоритмы', count: 6 },
  { title: 'System Design', count: 14 },
  { title: 'Платформа', count: 21 },
  { title: 'Сети', count: 10 },
  { title: 'ОС и Linux', count: 6 },
] as const

const PAIN_POINTS = [
  {
    n: '01',
    title: 'Знаете — но не можете рассказать',
    text: 'Интервьюер копает на три уровня: «а как это устроено внутри?» — и обрыв на втором.',
  },
  {
    n: '02',
    title: 'Прочитали — и забыли через неделю',
    text: 'Конспекты и плейлисты не работают без системы повторения. Мозгу нужны интервалы, а не марафоны.',
  },
  {
    n: '03',
    title: 'Лайвкодинг под таймером — ступор',
    text: 'Дома решаете за вечер, на секции — 25 минут и чужие глаза. Стресс снимается только повторением формата.',
  },
] as const

const FAQ_ITEMS = [
  {
    q: 'Чем это лучше конспекта вопросов с Хабра?',
    a: 'Конспект вы читаете один раз. Здесь каждый вопрос возвращается по расписанию SM-2, ответы разложены по уровням глубины, а лайвкодинг идёт под таймером с тестами — как на секции.',
  },
  {
    q: 'Сколько времени займёт подготовка?',
    a: '15–20 минут в день на вопросы и главы плюс 2–3 сессии лайвкодинга в неделю. При собеседовании через месяц-полтора — комфортный темп без марафонов по выходным.',
  },
  {
    q: 'Я мидл — мне рано?',
    a: 'Нет. Уровень middle в лестнице — отправная точка каждого ответа. Пройдёте программу — будете отвечать на senior-уровне: это и есть путь к следующему грейду и зарплатной вилке.',
  },
  {
    q: 'Это платно?',
    a: 'Сейчас — полностью бесплатно. Позже появится подписка на расширенную программу; всё открытое сейчас останется доступным.',
  },
] as const

// [ОТКЛ №2] ПЛЕЙСХОЛДЕРЫ — заменить реальными отзывами перед публикацией.
// Инициалы/должности абстрактные: без фамилий и компаний.
const SHOW_TESTIMONIALS = false
const PLACEHOLDER_TESTIMONIALS = [
  {
    initials: 'ДК',
    name: 'Дмитрий К.',
    role: 'оффер Senior Go',
    quote:
      '«На секции по Go спросили ровно то, что я повторял утром: hchan и вытеснение из буфера. Отвечал на уровне deep — интервьюер сам перешёл к следующей теме»',
  },
  {
    initials: 'АС',
    name: 'Артём С.',
    role: 'оффер Senior Go',
    quote:
      '«Главное — режим 25/5. Первые задачи заваливал по таймеру, к третьей неделе перестал паниковать. На реальном лайвкодинге было ощущение, что я это уже делал»',
  },
  {
    initials: 'МВ',
    name: 'Мария В.',
    role: 'оффер Senior Go',
    quote:
      '«Готовилась в декрете по 15 минут с телефона. Повторение само решало, что учить сегодня — не тратила ни минуты на планирование. Два оффера из трёх финалов»',
  },
] as const

const COMPANIES = ['Яндекс', 'Ozon', 'Авито', 'Т-Банк', 'СБЕР', 'VK'] as const

// Декоративные (некликабельные) кнопки самооценки лестницы.
const SELF_ASSESS = [
  { label: 'Снова', color: 'text-err', bg: 'bg-err-soft' },
  { label: 'Трудно', color: 'text-warn', bg: 'bg-warn-soft' },
  { label: 'Хорошо', color: 'text-ok', bg: 'bg-ok-soft' },
  { label: 'Легко', color: 'text-info', bg: 'bg-info-soft' },
] as const

function Overline({ children }: { children: ReactNode }) {
  return (
    <div className="font-mono text-[11px] font-semibold tracking-[0.1em] text-ink-3">{children}</div>
  )
}

export default function LandingPage() {
  const navigate = useNavigate()
  const [menuOpen, setMenuOpen] = useState(false)
  const heroCardRef = useRef<HTMLDivElement>(null)

  function scrollToSection(id: string) {
    setMenuOpen(false)
    document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }

  function scrollToHeroCard() {
    heroCardRef.current?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }

  return (
    <div className="goprep-landing min-h-svh overflow-x-hidden font-sans text-ink">
      {/* ================= Навбар — плавающая glass-пилюля ================= */}
      <div className="sticky top-0 z-20 flex flex-col items-center px-6 pt-[14px]">
        <div className="lp-glass-nav flex h-[52px] w-full max-w-[920px] items-center gap-[26px] rounded-full px-[26px]">
          <button
            type="button"
            onClick={() => scrollToSection('top')}
            className="text-[16px] font-semibold"
          >
            GoPrep<span className="text-ink-3">_</span>
          </button>

          <nav className="hidden gap-[22px] text-[13px] font-medium text-ink-2 md:flex">
            {NAV_LINKS.map((l) => (
              <button
                key={l.id}
                type="button"
                onClick={() => scrollToSection(l.id)}
                className="transition-colors duration-150 ease-out hover:text-ink"
              >
                {l.label}
              </button>
            ))}
          </nav>

          <div className="ml-auto hidden items-center gap-[10px] md:flex">
            <button
              type="button"
              onClick={() => navigate('/login')}
              className="text-[13px] font-medium text-ink-2 transition-colors duration-150 ease-out hover:text-ink"
            >
              Войти
            </button>
            <button
              type="button"
              onClick={() => navigate('/register')}
              className="rounded-full bg-ink px-[18px] py-2 text-[13px] font-medium text-bg transition-colors duration-150 ease-out hover:bg-accent-hover"
            >
              Начать бесплатно
            </button>
          </div>

          {/* Бургер — мобайл */}
          <button
            type="button"
            aria-label="Меню"
            aria-expanded={menuOpen}
            onClick={() => setMenuOpen((v) => !v)}
            className="ml-auto inline-flex size-9 items-center justify-center rounded-full text-ink transition-colors duration-150 ease-out hover:bg-accent-soft md:hidden"
          >
            {menuOpen ? <XIcon className="size-5" /> : <MenuIcon className="size-5" />}
          </button>
        </div>

        {/* Выпадающая панель меню (мобайл) */}
        {menuOpen && (
          <div className="lp-glass-nav mt-2 flex w-full max-w-[920px] flex-col gap-1 rounded-[20px] p-3 md:hidden">
            {NAV_LINKS.map((l) => (
              <button
                key={l.id}
                type="button"
                onClick={() => scrollToSection(l.id)}
                className="rounded-[12px] px-3 py-3 text-left text-[14px] font-medium text-ink-2 transition-colors duration-150 ease-out hover:bg-accent-soft hover:text-ink"
              >
                {l.label}
              </button>
            ))}
            <div className="mt-1 flex flex-col gap-2 border-t border-hairline pt-3">
              <button
                type="button"
                onClick={() => navigate('/login')}
                className="rounded-full border border-ink px-[18px] py-[10px] text-[14px] font-medium text-ink"
              >
                Войти
              </button>
              <button
                type="button"
                onClick={() => navigate('/register')}
                className="rounded-full bg-ink px-[18px] py-[11px] text-[14px] font-medium text-bg"
              >
                Начать бесплатно
              </button>
            </div>
          </div>
        )}
      </div>

      {/* ================= Hero ================= */}
      <section id="top" className="relative overflow-hidden">
        {/* Фон: два световых поля */}
        <div className="pointer-events-none absolute inset-0" aria-hidden="true">
          <div
            className="absolute -top-[140px] -right-[100px] h-[560px] w-[560px] rounded-full"
            style={{
              background:
                'radial-gradient(circle,rgba(190,178,160,.28),rgba(190,178,160,0) 65%)',
            }}
          />
          <div
            className="absolute -bottom-[200px] -left-[140px] h-[520px] w-[520px] rounded-full"
            style={{
              background:
                'radial-gradient(circle,rgba(170,180,195,.22),rgba(170,180,195,0) 65%)',
            }}
          />
        </div>

        <div className="relative mx-auto grid max-w-[1080px] grid-cols-1 items-center gap-10 px-6 pt-14 pb-16 md:px-10 lg:grid-cols-[1.05fr_0.95fr] lg:gap-[56px] lg:pt-[72px] lg:pb-[92px]">
          {/* Левая колонка */}
          <div className="flex flex-col gap-6">
            <Overline>ТРЕНАЖЁР СОБЕСЕДОВАНИЙ · SENIOR GOLANG</Overline>
            <h1 className="font-serif text-[38px] leading-[1.15] font-bold text-balance md:text-[50px]">
              Оффер senior-грейда — это не удача. Это подготовка.
            </h1>
            <p className="max-w-[460px] text-[16.5px] leading-[1.65] text-ink-2 text-pretty">
              Вопросы с реальных секций Яндекса, Ozon, Авито и Т-Банка. Интервальное повторение
              доводит ответы до автоматизма — на интервью вы говорите, а не вспоминаете.
            </p>
            <div className="flex flex-wrap items-center gap-3">
              <button
                type="button"
                onClick={() => navigate('/register')}
                className="rounded-full bg-ink px-7 py-[13px] text-[15px] font-medium text-bg transition-colors duration-150 ease-out hover:bg-accent-hover"
              >
                Начать бесплатно
              </button>
              <button
                type="button"
                onClick={scrollToHeroCard}
                className="lp-glass-btn rounded-full px-6 py-3 text-[15px] font-medium text-ink transition-colors duration-150 ease-out"
              >
                Попробовать вопрос ↓
              </button>
            </div>
            <div className="flex flex-wrap gap-7">
              {HERO_METRICS.map((m) => (
                <div key={m.label} className="flex flex-col gap-0.5">
                  <span className="font-mono text-[22px] font-semibold">{m.value}</span>
                  <span className="text-[11.5px] text-ink-3">{m.label}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Правая колонка — glass-карточка вопроса (интерактивная демка) */}
          <div className="relative">
            <div
              ref={heroCardRef}
              className="lp-glass-card flex scroll-mt-24 flex-col gap-[14px] rounded-[18px] p-[22px]"
            >
              <div className="flex items-center gap-2">
                <span className="rounded-full bg-accent-soft px-[10px] py-[3px] text-[11px] font-medium text-ink">
                  senior
                </span>
                <span className="rounded-full bg-warn-soft px-[10px] py-[3px] text-[11px] font-medium text-warn">
                  к повторению
                </span>
                <span className="ml-auto font-mono text-[10.5px] text-ink-3">12 / 40</span>
              </div>

              {/* [ОТКЛ] Текст вопроса hero захардкожен по ТЗ (чтение из закрытого канала). */}
              <div className="font-serif text-[19px] leading-[1.4] font-bold text-pretty">
                Что произойдёт при чтении из закрытого канала?
              </div>

              <div className="flex flex-col gap-1.5 border-t border-hairline pt-3">
                <span className="self-start rounded-full bg-tint-neutral px-2 py-0.5 text-[10.5px] font-medium text-ink-2">
                  middle
                </span>
                <div className="font-serif text-[14.5px] leading-[1.6]">
                  Чтение не блокируется и не паникует. Сначала отдаются буферизованные значения,
                  затем — нулевое значение типа и{' '}
                  <span className="rounded-[4px] bg-tint-neutral px-[5px] py-px font-mono text-[11.5px]">
                    ok == false
                  </span>
                  .
                </div>
              </div>

              {/* senior — за ссылкой-заглушкой на /register */}
              <div className="flex items-center gap-[10px] border-t border-hairline pt-3">
                <span className="rounded-full bg-accent-soft px-2 py-0.5 text-[10.5px] font-medium text-ink">
                  senior
                </span>
                <Link
                  to="/register"
                  className="text-[12.5px] font-medium text-ink underline underline-offset-[3px] transition-colors duration-150 ease-out hover:text-accent-hover"
                >
                  Раскрыть уровень senior →
                </Link>
              </div>

              {/* Кнопки самооценки — декоративные, некликабельные */}
              <div className="flex gap-1.5 border-t border-hairline pt-[14px]" aria-hidden="true">
                {SELF_ASSESS.map((b) => (
                  <span
                    key={b.label}
                    className={`pointer-events-none rounded-full px-3 py-1.5 text-[11.5px] font-medium select-none ${b.color} ${b.bg}`}
                  >
                    {b.label}
                  </span>
                ))}
              </div>
            </div>

            {/* Плавающая glass-пилюля таймера */}
            <div className="lp-glass-timer absolute -right-[14px] -bottom-[24px] flex items-center gap-[9px] rounded-full px-[18px] py-[9px]">
              <div className="size-[7px] rounded-full bg-ok" />
              <span className="font-mono text-[16px] font-semibold">18:42</span>
              <span className="text-[11px] text-ink-3">фокус 25/5</span>
            </div>
          </div>
        </div>

        {/* Полоса компаний */}
        <div className="relative border-t border-hairline">
          <div className="mx-auto flex max-w-[1080px] flex-wrap items-center gap-x-9 gap-y-3 px-6 py-[22px] md:px-10">
            <span className="font-mono text-[10.5px] font-semibold tracking-[0.08em] text-ink-3">
              ГОТОВИМ К ФОРМАТАМ СЕКЦИЙ
            </span>
            <div className="flex flex-wrap gap-x-[26px] gap-y-2 text-[13.5px] font-medium text-ink-2">
              {COMPANIES.map((c) => (
                <span key={c}>{c}</span>
              ))}
            </div>
          </div>
        </div>
      </section>

      {/* ================= Блок боли (тёмный) ================= */}
      <section className="border-t border-hairline bg-ink">
        <div className="mx-auto grid max-w-[1080px] grid-cols-1 items-center gap-10 px-6 py-16 md:grid-cols-[0.9fr_1.1fr] md:gap-[56px] md:px-10 md:py-16">
          <div className="flex flex-col gap-4">
            <div className="font-mono text-[11px] font-semibold tracking-[0.1em] text-[#7C7871]">
              ПОЧЕМУ ВАЛЯТСЯ СИЛЬНЫЕ
            </div>
            <h2 className="font-serif text-[30px] leading-[1.3] font-bold text-bg text-pretty">
              Вы пишете на Go пять лет. И всё равно плывёте на вопросе про закрытый канал.
            </h2>
            <p className="text-[15px] leading-[1.7] text-[#A7A29A] text-pretty">
              Работа и собеседование — разные виды спорта. На работе вы решаете задачи, на интервью —
              объясняете внутренности под таймером. Это тренируется. Отдельно.
            </p>
          </div>
          <div className="flex flex-col">
            {PAIN_POINTS.map((p, i) => (
              <div
                key={p.n}
                className={`flex gap-4 border-t border-[#3A3833] py-[18px] ${
                  i === PAIN_POINTS.length - 1 ? 'border-b' : ''
                }`}
              >
                <span className="shrink-0 font-mono text-[13px] font-semibold text-[#7C7871]">
                  {p.n}
                </span>
                <div className="flex flex-col gap-1">
                  <span className="text-[15px] font-semibold text-bg">{p.title}</span>
                  <span className="text-[13.5px] leading-[1.6] text-[#A7A29A]">{p.text}</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* ================= Как это работает ================= */}
      <section id="how" className="scroll-mt-24 border-t border-hairline">
        <div className="mx-auto flex max-w-[1080px] flex-col gap-11 px-6 py-16 md:px-10 md:py-[72px]">
          <div className="flex max-w-[640px] flex-col gap-[10px]">
            <Overline>КАК ЭТО РАБОТАЕТ</Overline>
            <h2 className="font-serif text-[28px] leading-[1.25] font-bold text-pretty md:text-[32px]">
              Готовьтесь как спортсмен: план, подходы, повторения
            </h2>
          </div>

          <div className="flex flex-col">
            {/* 01 — учебник */}
            <HowRow
              n="01"
              title="Закройте пробелы за 5–10 минут в день"
              text="Учебник со схемами по темам, которые реально спрашивают: планировщик, каналы изнутри, GC. Читается с телефона — в метро, в очереди, перед сном."
            >
              <div className="flex flex-col gap-2 rounded-[10px] border border-hairline bg-surface p-4">
                <div className="font-serif text-[14px] font-bold">
                  Каналы изнутри: hchan, очереди, блокировки
                </div>
                <div className="flex items-center gap-[10px]">
                  <div className="flex h-1 flex-1 gap-[2px] overflow-hidden rounded-[2px]">
                    {Array.from({ length: 9 }).map((_, i) => (
                      <div key={i} className={`flex-1 ${i < 4 ? 'bg-ink' : 'bg-hairline'}`} />
                    ))}
                  </div>
                  <span className="font-mono text-[10.5px] whitespace-nowrap text-ink-3">
                    гл. 5/9 · ~8 мин
                  </span>
                </div>
              </div>
            </HowRow>

            {/* 02 — лестница */}
            <HowRow
              n="02"
              title="Отвечайте глубже, чем ждёт интервьюер"
              text="Каждый вопрос — лестница из трёх ответов: middle → senior → deep. Отвечаете вслух, сверяетесь, честно оцениваете себя — и SM-2 вернёт вопрос ровно перед тем, как вы его забудете."
            >
              <div className="flex flex-col gap-[10px] rounded-[10px] border border-hairline bg-surface p-4">
                <div className="font-serif text-[14px] font-bold text-pretty">
                  Что происходит при отправке в закрытый канал?
                </div>
                <div className="flex flex-wrap gap-[5px]">
                  <span className="rounded-full bg-tint-neutral px-2 py-0.5 text-[10.5px] font-medium text-ink-2">
                    middle ✓
                  </span>
                  <span className="rounded-full bg-accent-soft px-2 py-0.5 text-[10.5px] font-medium text-ink">
                    senior ✓
                  </span>
                  <span className="rounded-full bg-staff-soft px-2 py-0.5 text-[10.5px] font-medium text-staff">
                    deep…
                  </span>
                </div>
                <div className="flex flex-wrap gap-[5px]" aria-hidden="true">
                  {SELF_ASSESS.map((b) => (
                    <span
                      key={b.label}
                      className={`rounded-full px-[10px] py-1 text-[11px] font-medium ${b.color} ${b.bg}`}
                    >
                      {b.label}
                    </span>
                  ))}
                </div>
              </div>
            </HowRow>

            {/* 03 — лайвкодинг */}
            <HowRow
              n="03"
              title="Прорешайте лайвкодинг в условиях секции"
              text="25 минут на задачу, тесты на сервере — как на реальном интервью. Застряли — подсказки по одной каждые 5 минут, потом разбор. Решённое возвращается через 7, 21 и 60 дней — написать по памяти."
              last
            >
              <div className="flex flex-col gap-2 rounded-[10px] bg-editor p-4">
                <div className="flex items-center justify-between">
                  <span className="font-mono text-[12px] font-medium text-editor-ink">lru.go</span>
                  <span className="font-mono text-[13px] font-semibold text-[#E9E6E0]">18:42</span>
                </div>
                <div className="font-mono text-[11.5px] leading-[1.7]">
                  <div className="text-[#7DBE8E]">PASS&nbsp;&nbsp;TestLRU_SetGet</div>
                  <div className="text-[#7DBE8E]">PASS&nbsp;&nbsp;TestLRU_Eviction</div>
                  <div className="text-[#E08A80]">FAIL&nbsp;&nbsp;TestLRU_TTLExpiry</div>
                </div>
              </div>
            </HowRow>
          </div>
        </div>
      </section>

      {/* ================= Отзывы ================= */}
      {SHOW_TESTIMONIALS && (
        <section id="testimonials" className="scroll-mt-24 border-t border-hairline bg-surface-2">
          <div className="mx-auto flex max-w-[1080px] flex-col gap-9 px-6 py-16 md:px-10 md:py-[72px]">
            <div className="flex flex-col gap-[10px]">
              <Overline>ОТЗЫВЫ</Overline>
              <h2 className="max-w-[560px] font-serif text-[28px] leading-[1.25] font-bold text-pretty md:text-[32px]">
                Проверено на чужих собеседованиях
              </h2>
            </div>
            <div className="grid grid-cols-1 gap-6 md:grid-cols-3">
              {PLACEHOLDER_TESTIMONIALS.map((t) => (
                <div
                  key={t.initials}
                  className="lp-glass-testimonial flex flex-col gap-[14px] rounded-[14px] p-6"
                >
                  <div className="font-serif text-[15.5px] leading-[1.6] italic text-pretty">
                    {t.quote}
                  </div>
                  <div className="mt-auto flex items-center gap-[10px]">
                    <div className="flex size-[34px] items-center justify-center rounded-full bg-hairline text-[12px] font-semibold text-ink-2">
                      {t.initials}
                    </div>
                    <div className="flex flex-col">
                      <span className="text-[13px] font-semibold">{t.name}</span>
                      <span className="text-[11.5px] text-ink-3">{t.role}</span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
            <div className="text-[11.5px] text-ink-3">
              Плейсхолдеры — заменить на реальные отзывы бета-пользователей.
            </div>
          </div>
        </section>
      )}

      {/* ================= Программа ================= */}
      <section id="program" className="scroll-mt-24 border-t border-hairline">
        <div className="mx-auto grid max-w-[1080px] grid-cols-1 gap-10 px-6 py-16 md:grid-cols-[0.9fr_1.1fr] md:gap-16 md:px-10 md:py-[72px]">
          <div className="flex flex-col gap-[14px]">
            <Overline>ПРОГРАММА</Overline>
            <h2 className="font-serif text-[28px] leading-[1.25] font-bold text-pretty md:text-[32px]">
              Только то, что спрашивают. Ничего «на всякий случай»
            </h2>
            <p className="text-[15px] leading-[1.65] text-ink-2 text-pretty">
              Программа собрана из реальных секций Go-собеседований больших компаний СНГ и
              обновляется по свежим интервью. Каждый вопрос — с лестницей из трёх ответов и ссылкой
              на главу учебника.
            </p>
            <button
              type="button"
              onClick={() => navigate('/register')}
              className="self-start rounded-full border border-ink px-5 py-[10px] text-[14px] font-medium text-ink transition-colors duration-150 ease-out hover:bg-accent-soft"
            >
              Смотреть все темы
            </button>
          </div>
          <div className="flex flex-col">
            {PROGRAM_SECTIONS.map((s, i) => (
              <div
                key={s.title}
                className={`flex items-center gap-[14px] border-t border-hairline py-[14px] ${
                  i === PROGRAM_SECTIONS.length - 1 ? 'border-b' : ''
                }`}
              >
                <span className="flex-1 text-[15px] font-medium">{s.title}</span>
                <span className="font-mono text-[12px] text-ink-3">
                  {s.count} {plural(s.count, ['вопрос', 'вопроса', 'вопросов'])}
                </span>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* ================= FAQ ================= */}
      <section id="faq" className="scroll-mt-24 border-t border-hairline">
        <div className="mx-auto flex max-w-[760px] flex-col gap-7 px-6 py-16 md:px-10 md:py-[72px]">
          <div className="flex flex-col gap-[10px]">
            <Overline>FAQ</Overline>
            <h2 className="font-serif text-[28px] leading-[1.25] font-bold md:text-[32px]">
              Частые вопросы
            </h2>
          </div>
          <div className="flex flex-col">
            {FAQ_ITEMS.map((f, i) => (
              <div
                key={f.q}
                className={`flex flex-col gap-2 border-t border-hairline py-[18px] ${
                  i === FAQ_ITEMS.length - 1 ? 'border-b' : ''
                }`}
              >
                <div className="text-[15px] font-semibold">{f.q}</div>
                <div className="text-[14.5px] leading-[1.65] text-ink-2">{f.a}</div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* ================= Финальный CTA (тёмный) + футер ================= */}
      <section className="border-t border-hairline bg-ink">
        <div className="mx-auto flex max-w-[1080px] flex-col items-center gap-5 px-6 py-[76px] text-center md:px-10">
          <h2 className="max-w-[640px] font-serif text-[30px] leading-[1.25] font-bold text-balance text-bg md:text-[36px]">
            Ваше следующее собеседование уже назначено. Вопрос в том, кто к нему готов.
          </h2>
          <p className="max-w-[440px] text-[15px] leading-[1.6] text-[#A7A29A]">
            Начните с 15 минут сегодня — система сама выстроит план на завтра.
          </p>
          <button
            type="button"
            onClick={() => navigate('/register')}
            className="rounded-full bg-bg px-[30px] py-[14px] text-[15px] font-medium text-ink transition-colors duration-150 ease-out hover:bg-accent-soft"
          >
            Начать бесплатно
          </button>
          {/* [ОТКЛ №1] Реальные числа: 84 вопроса и 22 задачи (в мокапе 156/24). */}
          <div className="text-[12px] text-[#7C7871]">
            Без карты · 84 вопроса и 22 задачи в бесплатной программе
          </div>
        </div>

        <div className="border-t border-[#3A3833]">
          <div className="mx-auto flex max-w-[1080px] flex-wrap items-center gap-x-6 gap-y-3 px-6 py-5 md:px-10">
            <div className="text-[14px] font-semibold text-bg">
              GoPrep<span className="text-[#7C7871]">_</span>
            </div>
            <div className="flex flex-wrap gap-x-5 gap-y-2 text-[12.5px] text-[#A7A29A]">
              <button type="button" onClick={() => scrollToSection('how')} className="hover:text-bg">
                Как это работает
              </button>
              <button
                type="button"
                onClick={() => scrollToSection('program')}
                className="hover:text-bg"
              >
                Программа
              </button>
              <button type="button" onClick={() => scrollToSection('faq')} className="hover:text-bg">
                FAQ
              </button>
              <span>Телеграм</span>
            </div>
            <div className="ml-auto font-mono text-[11px] text-[#7C7871]">© 2026 GoPrep</div>
          </div>
        </div>
      </section>
    </div>
  )
}

function HowRow({
  n,
  title,
  text,
  children,
  last = false,
}: {
  n: string
  title: string
  text: string
  children: ReactNode
  last?: boolean
}) {
  return (
    <div
      className={`grid grid-cols-1 gap-4 border-t border-hairline py-[26px] md:grid-cols-[64px_1fr_380px] md:items-center md:gap-6 ${
        last ? 'border-b' : ''
      }`}
    >
      <span className="font-mono text-[20px] font-semibold text-ink-3">{n}</span>
      <div className="flex flex-col gap-1.5">
        <span className="text-[18px] font-semibold">{title}</span>
        <span className="text-[14.5px] leading-[1.65] text-ink-2 text-pretty">{text}</span>
      </div>
      {children}
    </div>
  )
}
