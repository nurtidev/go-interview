package api

// SectionInfo is a static catalog entry describing one section of the bank.
type SectionInfo struct {
	ID          string
	Title       string
	Description string
}

// Sections is the hardcoded, ordered catalog of question sections.
var Sections = []SectionInfo{
	{
		ID:          "go-internals",
		Title:       "Go изнутри",
		Description: "Устройство рантайма: планировщик горутин, модель памяти, сборщик мусора и интерфейсы.",
	},
	{
		ID:          "concurrency",
		Title:       "Конкурентность",
		Description: "Горутины и каналы, пакет sync, паттерны и типичные ошибки конкурентного кода.",
	},
	{
		ID:          "algorithms",
		Title:       "Алгоритмы",
		Description: "Структуры данных, оценка сложности и задачи, которые дают на алгоритмических секциях.",
	},
	{
		ID:          "system-design",
		Title:       "System Design",
		Description: "Проектирование распределённых и highload-систем: масштабирование, консистентность, очереди.",
	},
	{
		ID:          "platform",
		Title:       "Платформа и инфраструктура",
		Description: "Docker и Kubernetes, базы данных, наблюдаемость, сети и CI/CD для сервисов на Go.",
	},
	{
		ID:          "networks",
		Title:       "Сети",
		Description: "TCP/HTTP/TLS/DNS, балансировка нагрузки и сетевые ловушки Go-сервисов.",
	},
	{
		ID:          "os",
		Title:       "ОС и Linux",
		Description: "Процессы и память ОС, epoll, cgroups — фундамент под рантайм Go и контейнеры.",
	},
	{
		ID:          "private",
		Title:       "Приватный тренажёр",
		Description: "Личные задачи и вопросы.",
	},
}
