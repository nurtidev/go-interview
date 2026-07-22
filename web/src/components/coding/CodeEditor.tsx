import { useEffect, useRef } from 'react'
import { EditorState, Prec } from '@codemirror/state'
import {
  EditorView,
  drawSelection,
  highlightActiveLine,
  highlightActiveLineGutter,
  keymap,
  lineNumbers,
} from '@codemirror/view'
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands'
import {
  HighlightStyle,
  StreamLanguage,
  bracketMatching,
  indentUnit,
  syntaxHighlighting,
} from '@codemirror/language'
import { go } from '@codemirror/legacy-modes/mode/go'
import { sql } from '@codemirror/lang-sql'
import { tags as t } from '@lezer/highlight'
import type { CodingKind } from '@/lib/api'

// Подсветка «графит» (спека 1i): ключевые слова #8FA5C8, имена функций #D8A44A,
// комментарии #7A8087, обычный текст --editor-ink #D6D9DE. Палитра диаграммного тёмного острова.
const highlightStyle = HighlightStyle.define([
  {
    tag: [
      t.keyword,
      t.controlKeyword,
      t.moduleKeyword,
      t.operatorKeyword,
      t.definitionKeyword,
    ],
    color: '#8FA5C8',
  },
  { tag: [t.typeName, t.number, t.bool, t.atom, t.null], color: '#8FA5C8' },
  { tag: [t.string, t.special(t.string), t.character], color: '#D8A44A' },
  { tag: [t.function(t.variableName), t.function(t.propertyName)], color: '#D8A44A' },
  { tag: [t.comment, t.lineComment, t.blockComment], color: '#7A8087', fontStyle: 'italic' },
  { tag: [t.operator, t.punctuation, t.bracket], color: '#A7A29A' },
  { tag: [t.propertyName, t.variableName], color: '#D6D9DE' },
  { tag: [t.invalid], color: '#E08A80' },
])

const editorTheme = EditorView.theme(
  {
    '&': {
      backgroundColor: '#1E2023',
      color: '#D6D9DE',
      fontSize: '13.5px',
    },
    '&.cm-focused': { outline: 'none' },
    '.cm-scroller': {
      fontFamily:
        "'JetBrains Mono', ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Consolas, monospace",
      lineHeight: '1.75',
      overflow: 'auto',
      maxHeight: '460px',
    },
    '.cm-content': { padding: '12px 0', caretColor: '#D6D9DE' },
    '.cm-gutters': {
      backgroundColor: '#1E2023',
      color: '#5C6167',
      border: 'none',
      borderRight: '1px solid #33373D',
    },
    '.cm-lineNumbers .cm-gutterElement': { padding: '0 12px 0 14px' },
    '.cm-activeLine': { backgroundColor: 'rgba(255,255,255,0.03)' },
    '.cm-activeLineGutter': {
      backgroundColor: 'rgba(255,255,255,0.03)',
      color: '#8B9096',
    },
    '.cm-cursor, .cm-dropCursor': { borderLeftColor: '#D6D9DE' },
    '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection':
      {
        backgroundColor: 'rgba(214,217,222,0.16)',
      },
    '.cm-selectionMatch': { backgroundColor: 'rgba(214,217,222,0.10)' },
    '.cm-matchingBracket, &.cm-focused .cm-matchingBracket': {
      backgroundColor: 'rgba(214,217,222,0.16)',
      color: 'inherit',
    },
  },
  { dark: true },
)

interface CodeEditorProps {
  value: string
  onChange: (value: string) => void
  language: CodingKind
  className?: string
  /** Вызывается по Cmd+Enter / Ctrl+Enter — тот же обработчик, что и у кнопки «Запустить». */
  onRun?: () => void
}

/**
 * Управляемый редактор на CodeMirror 6. Создаётся один раз на монтирование
 * (язык фиксирован для задачи — страница задачи ремонтируется через key={slug}).
 * Внешние изменения value (например, «Сбросить к заготовке») синхронизируются
 * отдельным эффектом без петли обратной связи.
 */
export function CodeEditor({ value, onChange, language, className, onRun }: CodeEditorProps) {
  const hostRef = useRef<HTMLDivElement | null>(null)
  const viewRef = useRef<EditorView | null>(null)
  const onChangeRef = useRef(onChange)
  onChangeRef.current = onChange
  const onRunRef = useRef(onRun)
  onRunRef.current = onRun

  useEffect(() => {
    const host = hostRef.current
    if (!host) return

    const langExtension = language === 'go' ? StreamLanguage.define(go) : sql()

    const state = EditorState.create({
      doc: value,
      extensions: [
        lineNumbers(),
        highlightActiveLine(),
        highlightActiveLineGutter(),
        drawSelection(),
        history(),
        bracketMatching(),
        indentUnit.of('    '),
        Prec.high(
          keymap.of([
            {
              key: 'Mod-Enter',
              run: () => {
                onRunRef.current?.()
                return true
              },
            },
          ]),
        ),
        keymap.of([...defaultKeymap, ...historyKeymap, indentWithTab]),
        langExtension,
        syntaxHighlighting(highlightStyle),
        editorTheme,
        EditorView.updateListener.of((update) => {
          if (update.docChanged) onChangeRef.current(update.state.doc.toString())
        }),
      ],
    })

    const view = new EditorView({ state, parent: host })
    viewRef.current = view

    return () => {
      view.destroy()
      viewRef.current = null
    }
    // value читается только для стартового документа; последующая
    // синхронизация — в эффекте ниже. language фиксирован на время жизни.
  }, [language]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    const view = viewRef.current
    if (!view) return
    const current = view.state.doc.toString()
    if (value !== current) {
      view.dispatch({ changes: { from: 0, to: current.length, insert: value } })
    }
  }, [value])

  return <div ref={hostRef} className={className} />
}
