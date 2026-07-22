import ReactMarkdown, { type Components } from 'react-markdown'
import rehypeRaw from 'rehype-raw'
import { cn } from '@/lib/utils'

const FONT_CONTENT = 'var(--font-content, Georgia, serif)'
const FONT_CODE = 'var(--font-code, ui-monospace, monospace)'
const FONT_UI = 'var(--font-ui, system-ui, sans-serif)'

const components: Components = {
  // Схемы из markdown оборачиваем в figure с автонумерацией «Рис. N»
  // (CSS-счётчик на контейнере Markdown, инкремент — на каждой figure).
  svg: ({ node: _node, ...props }) => (
    <figure className="my-6 [counter-increment:figure]">
      <svg
        className="block h-auto w-full max-w-[600px] rounded-[10px] border border-[var(--hairline)] bg-[var(--surface)]"
        {...props}
      />
      <figcaption
        style={{ fontFamily: FONT_UI }}
        className="mt-2 text-center text-[12px] text-[var(--ink-3)] before:[content:'Рис._'_counter(figure)]"
      />
    </figure>
  ),
  code: ({ className, children, ...props }) => (
    <code
      style={{ fontFamily: FONT_CODE }}
      className={cn(
        'rounded-[4px] bg-[var(--accent-soft)] px-1.5 py-0.5 text-[0.76em] text-[var(--ink)]',
        className,
      )}
      {...props}
    >
      {children}
    </code>
  ),
  pre: ({ children, ...props }) => (
    <pre
      style={{ fontFamily: FONT_CODE }}
      className="-mx-4 mb-6 overflow-x-auto rounded-[var(--r-md)] bg-[var(--accent-soft)] p-4 text-[14px] leading-relaxed text-[var(--ink)] sm:-mx-6 [&>code]:bg-transparent [&>code]:p-0 [&>code]:text-[14px]"
      {...props}
    >
      {children}
    </pre>
  ),
  a: ({ children, ...props }) => (
    <a
      className="text-[var(--accent)] underline underline-offset-2 transition-colors hover:text-[var(--accent-hover)]"
      target="_blank"
      rel="noreferrer"
      {...props}
    >
      {children}
    </a>
  ),
  h1: (props) => (
    <h1 style={{ fontFamily: FONT_CONTENT }} className="mt-8 mb-4 text-2xl font-bold text-[var(--ink)]" {...props} />
  ),
  h2: (props) => (
    <h2 style={{ fontFamily: FONT_CONTENT }} className="mt-8 mb-4 text-xl font-bold text-[var(--ink)]" {...props} />
  ),
  h3: (props) => (
    <h3 style={{ fontFamily: FONT_CONTENT }} className="mt-6 mb-3 text-lg font-bold text-[var(--ink)]" {...props} />
  ),
  p: (props) => <p className="mb-[1.5em] last:mb-0" {...props} />,
  ul: (props) => <ul className="mb-[1.5em] ml-6 list-disc space-y-2" {...props} />,
  ol: (props) => <ol className="mb-[1.5em] ml-6 list-decimal space-y-2" {...props} />,
  li: (props) => <li {...props} />,
  blockquote: (props) => (
    <blockquote className="mb-[1.5em] border-l-2 border-[var(--hairline)] pl-4 text-[var(--ink-2)] italic" {...props} />
  ),
  table: (props) => (
    <div className="mb-[1.5em] overflow-x-auto rounded-[var(--r-md)] border border-[var(--hairline)]">
      <table style={{ fontFamily: FONT_UI }} className="w-full border-collapse text-[14px]" {...props} />
    </div>
  ),
  th: (props) => (
    <th
      className="border-b border-[var(--hairline)] bg-[var(--accent-soft)] px-3 py-2 text-left text-[var(--ink-2)] font-semibold"
      {...props}
    />
  ),
  td: (props) => (
    <td className="border-t border-[var(--hairline)] px-3 py-2 text-[var(--ink)]" {...props} />
  ),
  strong: (props) => <strong className="font-bold text-[var(--ink)]" {...props} />,
  em: (props) => <em className="italic" {...props} />,
}

const SIZE_CLASS = {
  lg: 'text-[20px] leading-[1.7]',
  md: 'text-[17px] leading-[1.65]',
  sm: 'text-[16px] leading-[1.65]',
} as const

export function Markdown({
  children,
  size = 'lg',
}: {
  children: string
  size?: keyof typeof SIZE_CLASS
}) {
  return (
    <div
      style={{ fontFamily: FONT_CONTENT }}
      className={cn('text-[var(--ink)] [counter-reset:figure] [text-wrap:pretty]', SIZE_CLASS[size])}
    >
      <ReactMarkdown components={components} rehypePlugins={[rehypeRaw]}>
        {children}
      </ReactMarkdown>
    </div>
  )
}
